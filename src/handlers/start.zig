const std = @import("std");
const posix = std.posix;

fn getPackageJsonScript(allocator: std.mem.Allocator, script_name: []const u8) ?[]const u8 {
    const cwd = std.fs.cwd();
    const file = cwd.openFile("package.json", .{}) catch return null;
    defer file.close();

    const content = file.readToEndAlloc(allocator, 1024 * 1024) catch return null;
    defer allocator.free(content);

    var parsed = std.json.parseFromSlice(std.json.Value, allocator, content, .{}) catch return null;
    defer parsed.deinit();

    const root = parsed.value.object;
    const scripts = root.get("scripts") orelse return null;

    if (scripts != .object) return null;

    const script = scripts.object.get(script_name) orelse return null;

    if (script != .string) return null;

    return allocator.dupe(u8, script.string) catch return null;
}

pub fn handleStart(args: *std.process.ArgIterator) void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const flag = args.next() orelse {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m missing --name flag\n", .{});
        return;
    };

    if (!std.mem.eql(u8, flag, "--name")) {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m expected --name flag\n", .{});
        return;
    }

    const name = args.next() orelse {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m missing process name\n", .{});
        return;
    };

    var script_name: []const u8 = "start";
    var command_opt: ?[]const u8 = null;
    var follow_logs: bool = false;

    while (args.next()) |arg| {
        if (std.mem.eql(u8, arg, "--script")) {
            script_name = args.next() orelse {
                std.debug.print("\x1b[31m[-] Error:\x1b[0m missing script name after --script\n", .{});
                return;
            };
        } else if (std.mem.eql(u8, arg, "-f") or std.mem.eql(u8, arg, "--follow")) {
            follow_logs = true;
        } else {
            command_opt = arg;
        }
    }

    const command = if (command_opt) |cmd| cmd else blk: {
        if (getPackageJsonScript(allocator, script_name)) |script_cmd| {
            std.debug.print("\x1b[90m[*] Info: Found script in package.json:\x1b[0m \x1b[36m{s}\x1b[0m\n", .{script_name});
            break :blk script_cmd;
        } else {
            std.debug.print("\x1b[31m[-] Error:\x1b[0m no command provided and no package.json script found\n\n", .{});
            std.debug.print("\x1b[1mUsage:\x1b[0m\n  zpm start --name <name> [-f | --follow] [--script <script>] [command]\n", .{});
            return;
        }
    };
    defer if (command_opt == null) allocator.free(command);

    const client_fd = posix.socket(posix.AF.UNIX, posix.SOCK.STREAM, 0) catch {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m failed to initialize ipc socket channel\n", .{});
        return;
    };
    defer posix.close(client_fd);

    const socket_path = "/tmp/zpm.sock";
    var addr = posix.sockaddr.un{ .path = undefined };
    std.mem.copyForwards(u8, &addr.path, socket_path);
    addr.path[socket_path.len] = 0;

    const sock_len = @as(posix.socklen_t, @intCast(@offsetOf(posix.sockaddr.un, "path") + socket_path.len + 1));
    posix.connect(client_fd, @ptrCast(&addr), sock_len) catch {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m cannot connect to zpmd daemon. is it running?\n", .{});
        return;
    };

    const payload = std.fmt.allocPrint(allocator, "start\n{s}\n{s}\n", .{ name, command }) catch return;
    defer allocator.free(payload);

    _ = posix.write(client_fd, payload) catch {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m failed sending transmission packet to daemon\n", .{});
        return;
    };

    var incoming_response_buffer: [2048]u8 = undefined;
    const response_bytes = posix.read(client_fd, &incoming_response_buffer) catch {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m error attempting to receive response package from daemon\n", .{});
        return;
    };

    var response_lines = std.mem.splitSequence(u8, incoming_response_buffer[0..response_bytes], "\n");
    const status_header = response_lines.next() orelse "error";

    if (std.mem.eql(u8, status_header, "error")) {
        const message = response_lines.next() orelse "unknown server internal failure";
        std.debug.print("\x1b[31m[-] Daemon Refusal:\x1b[0m {s}\n", .{message});
        return;
    }

    const assigned_pid_str = response_lines.next() orelse "0";

    std.debug.print("\n\x1b[1;32m[+] Process successfully started\x1b[0m\n", .{});
    std.debug.print("\x1b[90m┌──────────────────────────────────────────────┐\x1b[0m\n", .{});
    std.debug.print("\x1b[90m│\x1b[0m  \x1b[1;34mName\x1b[0m     :  {s}\n", .{name});
    std.debug.print("\x1b[90m│\x1b[0m  \x1b[1;35mPID\x1b[0m      :  \x1b[33m{s}\x1b[0m\n", .{assigned_pid_str});
    std.debug.print("\x1b[90m│\x1b[0m  \x1b[1;36mCommand\x1b[0m  :  \x1b[32m{s}\x1b[0m\n", .{command});
    std.debug.print("\x1b[90m│\x1b[0m  \x1b[1;33mMode\x1b[0m     :  {s}\n", .{if (follow_logs) "Attached (Following Logs)" else "Detached (Background)"});
    std.debug.print("\x1b[90m└──────────────────────────────────────────────┘\x1b[0m\n\n", .{});

    if (follow_logs) {
        std.debug.print("\x1b[90m[*] Attaching to stream logs... Press Ctrl+C to detach without killing process.\x1b[0m\n\n", .{});

        const log_filename = std.fmt.allocPrint(allocator, "logs/{s}.log", .{name}) catch return;
        defer allocator.free(log_filename);

        std.time.sleep(std.time.ns_per_ms * 100);

        var file = while (true) {
            if (std.fs.cwd().openFile(log_filename, .{})) |f| {
                break f;
            } else |_| {
                std.time.sleep(std.time.ns_per_ms * 50);
            }
        };
        defer file.close();

        var buf: [4096]u8 = undefined;
        while (true) {
            const bytes_read = file.read(&buf) catch |err| {
                std.debug.print("\n\x1b[31m[-] Stream detached: {}\x1b[0m\n", .{err});
                break;
            };

            if (bytes_read > 0) {
                _ = std.io.getStdOut().write(buf[0..bytes_read]) catch break;
            } else {
                std.time.sleep(std.time.ns_per_ms * 100);
            }
        }
    }
}

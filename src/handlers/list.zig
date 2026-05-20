const std = @import("std");
const posix = std.posix;

pub fn handleList() void {
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

    _ = posix.write(client_fd, "list\n") catch return;

    var response_buffer: [8192]u8 = undefined;
    const bytes_read = posix.read(client_fd, &response_buffer) catch return;
    if (bytes_read == 0) return;

    var lines = std.mem.splitSequence(u8, response_buffer[0..bytes_read], "\n");
    const status_header = lines.next() orelse "error";

    if (std.mem.eql(u8, status_header, "error")) {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m server refused list layout request\n", .{});
        return;
    }

    std.debug.print("\n\x1b[1;36m┌─ zpm active process dashboard ──────────────────────────────────────────────┐\x1b[0m\n", .{});
    std.debug.print("\x1b[1;36m│\x1b[0m \x1b[1;37m{s:<12} {s:<10} {s:<8} {s:<8} {s:<10} {s:<22}\x1b[0m \x1b[1;36m│\x1b[0m\n", .{
        "NAME", "STATUS", "PID", "CPU %", "MEM (MB)", "COMMAND",
    });
    std.debug.print("\x1b[1;36m├──────────────────────────────────────────────────────────────────────────────┤\x1b[0m\n", .{});

    var row_count: usize = 0;

    while (lines.next()) |line| {
        if (line.len == 0) continue;
        row_count += 1;

        var tokens = std.mem.splitSequence(u8, line, "|");
        const name = tokens.next() orelse "unknown";
        const status = tokens.next() orelse "stopped";
        const pid_str = tokens.next() orelse "-";
        const cpu_str = tokens.next() orelse "0.0";
        const mem_str = tokens.next() orelse "0.0";
        const command = tokens.next() orelse "-";

        const status_color = if (std.mem.eql(u8, status, "running")) "\x1b[32m" else "\x1b[31m";

        var truncate_cmd: [22]u8 = undefined;
        const formatted_cmd = if (command.len > 22) blk: {
            std.mem.copyForwards(u8, truncate_cmd[0..19], command[0..19]);
            std.mem.copyForwards(u8, truncate_cmd[19..22], "...");
            break :blk &truncate_cmd;
        } else command;

        std.debug.print("\x1b[1;36m│\x1b[0m {s:<12} {s}{s:<10}\x1b[0m {s:<8} {s:<8} {s:<10} {s:<22} \x1b[1;36m│\x1b[0m\n", .{
            name,
            status_color,
            status,
            pid_str,
            cpu_str,
            mem_str,
            formatted_cmd,
        });
    }

    if (row_count == 0) {
        std.debug.print("\x1b[1;36m│\x1b[0m \x1b[90m{s:<76}\x1b[0m \x1b[1;36m│\x1b[0m\n", .{" No microservices registered inside daemon instance stack."});
    }

    std.debug.print("\x1b[1;36m└──────────────────────────────────────────────────────────────────────────────┘\x1b[0m\n\n", .{});
}

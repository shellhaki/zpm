const std = @import("std");
const posix = std.posix;

fn isDaemonActive() bool {
    const client_fd = posix.socket(posix.AF.UNIX, posix.SOCK.STREAM, 0) catch return false;
    defer posix.close(client_fd);
    const socket_path = "/tmp/zpm.sock";
    var addr = posix.sockaddr.un{ .path = undefined };
    std.mem.copyForwards(u8, &addr.path, socket_path);
    addr.path[socket_path.len] = 0;
    const sock_len = @as(posix.socklen_t, @intCast(@offsetOf(posix.sockaddr.un, "path") + socket_path.len + 1));
    posix.connect(client_fd, @ptrCast(&addr), sock_len) catch return false;
    return true;
}

fn sendDaemonSignal(signal: []const u8) !bool {
    const client_fd = try posix.socket(posix.AF.UNIX, posix.SOCK.STREAM, 0);
    defer posix.close(client_fd);
    const socket_path = "/tmp/zpm.sock";
    var addr = posix.sockaddr.un{ .path = undefined };
    std.mem.copyForwards(u8, &addr.path, socket_path);
    addr.path[socket_path.len] = 0;
    const sock_len = @as(posix.socklen_t, @intCast(@offsetOf(posix.sockaddr.un, "path") + socket_path.len + 1));
    try posix.connect(client_fd, @ptrCast(&addr), sock_len);
    _ = try posix.write(client_fd, signal);
    var buf: [128]u8 = undefined;
    const bytes = try posix.read(client_fd, &buf);
    return std.mem.startsWith(u8, buf[0..bytes], "success");
}

fn startBackgroundDaemon(allocator: std.mem.Allocator) !void {
    const self_exe_path = try std.fs.selfExePathAlloc(allocator);
    defer allocator.free(self_exe_path);

    const binary_dir = std.fs.path.dirname(self_exe_path) orelse return error.NoDirName;
    const daemon_absolute_path = try std.fs.path.join(allocator, &[_][]const u8{ binary_dir, "zpmd" });
    defer allocator.free(daemon_absolute_path);

    const pid = try posix.fork();
    if (pid == 0) {
        if (std.os.linux.setsid() == -1) posix.exit(1);
        const dev_null = try std.fs.openFileAbsolute("/dev/null", .{ .mode = .read_write });
        try posix.dup2(dev_null.handle, posix.STDIN_FILENO);
        try posix.dup2(dev_null.handle, posix.STDOUT_FILENO);
        try posix.dup2(dev_null.handle, posix.STDERR_FILENO);
        dev_null.close();
        var env_map = std.process.getEnvMap(allocator) catch posix.exit(1);
        defer env_map.deinit();
        const argv = [_][]const u8{daemon_absolute_path};
        _ = std.process.execve(allocator, &argv, &env_map) catch posix.exit(127);
        unreachable;
    }
}

pub fn handleDaemon(args: *std.process.ArgIterator) void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const sub_cmd = args.next() orelse {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m Missing daemon action (start, stop, reload)\n", .{});
        return;
    };

    if (std.mem.eql(u8, sub_cmd, "start")) {
        if (isDaemonActive()) {
            std.debug.print("\x1b[33m[*] Info:\x1b[0m Daemon engine is already running.\n", .{});
            return;
        }
        startBackgroundDaemon(allocator) catch {
            std.debug.print("\x1b[31m[-] Error:\x1b[0m Failed to launch daemon background session\n", .{});
            return;
        };
        std.debug.print("\x1b[1;32m[+] Daemon process successfully initialized in background\x1b[0m\n", .{});
    } else if (std.mem.eql(u8, sub_cmd, "stop")) {
        if (!isDaemonActive()) {
            std.debug.print("\x1b[33m[*] Info:\x1b[0m Daemon is already offline.\n", .{});
            return;
        }
        if (sendDaemonSignal("stop_daemon\n") catch false) {
            std.debug.print("\x1b[1;32m[+] Daemon successfully stopped\x1b[0m\n", .{});
        } else {
            std.debug.print("\x1b[31m[-] Error:\x1b[0m Shutdown transmission failed\n", .{});
        }
    } else if (std.mem.eql(u8, sub_cmd, "reload")) {
        if (isDaemonActive()) {
            _ = sendDaemonSignal("stop_daemon\n") catch false;
            std.time.sleep(std.time.ns_per_ms * 300);
        }
        startBackgroundDaemon(allocator) catch {
            std.debug.print("\x1b[31m[-] Error:\x1b[0m Failed to restart daemon\n", .{});
            return;
        };
        std.debug.print("\x1b[1;32m[+] Daemon safely reloaded in background\x1b[0m\n", .{});
    } else {
        std.debug.print("\x1b[31m[-] Error:\x1b[0m Unknown daemon action: {s}\n", .{sub_cmd});
    }
}

const std = @import("std");
const start = @import("handlers/start.zig");
const stop = @import("handlers/stop.zig");
const reg = @import("registry.zig");
const list = @import("handlers/list.zig");
const purge = @import("handlers/purge.zig");
const daemon = @import("handlers/daemon.zig");
const posix = std.posix;

fn isDaemonRunning() bool {
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

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    var args = try std.process.argsWithAllocator(allocator);
    defer args.deinit();
    _ = args.next();

    const cmd = args.next() orelse {
        std.debug.print("Invalid usage\n", .{});
        return;
    };

    const is_daemon_cmd = std.mem.eql(u8, cmd, "daemon");

    if (!is_daemon_cmd and !isDaemonRunning()) {
        std.debug.print("\n\x1b[31m[-] Connection Failure\x1b[0m\n", .{});
        std.debug.print("\x1b[90m┌──────────────────────────────────────────────┐\x1b[0m\n", .{});
        std.debug.print("\x1b[90m│\x1b[0m  The zpmd daemon engine is not running.       \n", .{});
        std.debug.print("\x1b[90m│\x1b[0m  Please execute 'zpm daemon start' first.     \n", .{});
        std.debug.print("\x1b[90m└──────────────────────────────────────────────┘\x1b[0m\n\n", .{});
        return;
    }

    //try reg.load(allocator);

    if (std.mem.eql(u8, cmd, "start")) {
        start.handleStart(&args);
    } else if (std.mem.eql(u8, cmd, "stop")) {
        stop.handleStop(&args);
    } else if (std.mem.eql(u8, cmd, "list")) {
        list.handleList();
    } else if (std.mem.eql(u8, cmd, "purge")) {
        purge.handlePurge(&args);
    } else if (is_daemon_cmd) {
        daemon.handleDaemon(&args);
    } else {
        std.debug.print("unknown command: {s}\n", .{cmd});
    }
}

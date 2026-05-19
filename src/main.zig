const std = @import("std");
const start = @import("handlers/start.zig");
const stop = @import("handlers/stop.zig");
const reg = @import("registry.zig");
const list = @import("handlers/list.zig");
const purge = @import("handlers/purge.zig");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();

    const allocator = gpa.allocator();
    try reg.load(allocator);

    var args = try std.process.argsWithAllocator(allocator);
    defer args.deinit();

    _ = args.next();

    const cmd = args.next() orelse {
        std.debug.print(
            "Invalid usage\n",
            .{},
        );
        return;
    };

    if (std.mem.eql(u8, cmd, "start")) {
        start.handleStart(&args);
    } else if (std.mem.eql(u8, cmd, "stop")) {
        stop.handleStop(&args);
    } else if (std.mem.eql(u8, cmd, "list")) {
        list.handleList();
    } else if (std.mem.eql(u8, cmd, "purge")) {
        purge.handlePurge(&args);
        std.debug.print(
            "unknown command: {s}\n",
            .{cmd},
        );
    }
}

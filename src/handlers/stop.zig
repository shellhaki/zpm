const std = @import("std");
const registry = @import("../registry.zig");

pub fn handleStop(args: *std.process.ArgIterator) void {
    const name = args.next() orelse {
        std.debug.print("missing process name\n", .{});
        return;
    };

    registry.remove(name);

    std.debug.print("process marked as stopped\n", .{});
    std.debug.print("name: {s}\n", .{name});
}

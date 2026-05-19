const std = @import("std");
const registry = @import("../registry.zig");

pub fn handlePurge(args: *std.process.ArgIterator) void {
    const name = args.next() orelse {
        std.debug.print("missing process name\n", .{});
        return;
    };

    const removed = registry.purge(name);

    if (removed) {
        std.debug.print("process purged\n", .{});
        std.debug.print("name: {s}\n", .{name});
    } else {
        std.debug.print("process not found: {s}\n", .{name});
    }
}

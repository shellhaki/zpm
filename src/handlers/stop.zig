const std = @import("std");
const registry = @import("../registry.zig");

pub fn handleStop(args: *std.process.ArgIterator) void {
    const name = args.next() orelse {
        std.debug.print("missing process name\n", .{});
        return;
    };

    if (!registry.stopProcess(name)) {
        std.debug.print("process not found or could not be stopped: {s}\n", .{name});
        return;
    }

    registry.remove(name);

    std.debug.print("process marked as stopped\n", .{});
    std.debug.print("name: {s}\n", .{name});
}

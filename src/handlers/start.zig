const std = @import("std");
const registry = @import("../registry.zig");

pub fn handleStart(args: *std.process.ArgIterator) void {
    const flag = args.next() orelse {
        std.debug.print("missing --name flag\n", .{});
        return;
    };

    if (!std.mem.eql(u8, flag, "--name")) {
        std.debug.print("expected --name flag\n", .{});
        return;
    }

    const name = args.next() orelse {
        std.debug.print("missing process name\n", .{});
        return;
    };

    const command = args.next() orelse {
        std.debug.print("missing command\n", .{});
        return;
    };

    registry.add(name, command, 0);

    std.debug.print("process started\n", .{});
    std.debug.print("name: {s}\n", .{name});
    std.debug.print("command: {s}\n", .{command});
}

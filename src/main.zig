const std = @import("std");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();

    const allocator = gpa.allocator();

    var args = try std.process.argsWithAllocator(allocator);
    defer args.deinit();

    // skip program name
    _ = args.next();

    const cmd = args.next() orelse {
        std.debug.print(
            "Invalid usage\n",
            .{},
        );
        return;
    };

    if (std.mem.eql(u8, cmd, "start")) {
        std.debug.print(
            "process started\n",
            .{},
        );
    } else if (std.mem.eql(u8, cmd, "stop")) {
        std.debug.print(
            "process stopped\n",
            .{},
        );
    } else if (std.mem.eql(u8, cmd, "list")) {
        std.debug.print(
            "listing processes...\n",
            .{},
        );
    } else {
        std.debug.print(
            "unknown command: {s}\n",
            .{cmd},
        );
    }
}

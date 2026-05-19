const std = @import("std");
const registry = @import("../registry.zig");

pub fn handleList() void {
    const processes = registry.getAll();

    if (processes.len == 0) {
        std.debug.print("no processes running\n", .{});
        return;
    }

    for (processes) |p| {
        const status = switch (p.status) {
            .running => "running",
            .stopped => "stopped",
        };

        std.debug.print("{s}  [{s}]  {s}\n", .{
            p.name,
            status,
            p.command,
        });
    }
}

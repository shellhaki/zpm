const std = @import("std");
const builtin = @import("builtin");
const registry = @import("registry.zig");
const posix = std.posix;

extern fn setsid() c_int;

fn currentEnvironment() [*:null]const ?[*:0]const u8 {
    return @as([*:null]const ?[*:0]const u8, @ptrCast(std.os.environ.ptr));
}

fn detachSession() void {
    switch (builtin.os.tag) {
        .linux => _ = std.os.linux.setsid(),
        .macos => _ = setsid(),
        else => {},
    }
}

const Command = union(enum) {
    start: struct { name: []const u8, command: []const u8 },
    stop: []const u8,
    purge: []const u8,
    list,
};

fn parseCommand(allocator: std.mem.Allocator, data: []const u8) !Command {
    var lines = std.mem.splitSequence(u8, data, "\n");
    const cmd_type = lines.next() orelse return error.InvalidCommand;

    if (std.mem.eql(u8, cmd_type, "start")) {
        const name = lines.next() orelse return error.InvalidCommand;
        const command = lines.next() orelse return error.InvalidCommand;
        return Command{ .start = .{
            .name = try allocator.dupe(u8, name),
            .command = try allocator.dupe(u8, command),
        } };
    } else if (std.mem.eql(u8, cmd_type, "stop")) {
        const name = lines.next() orelse return error.InvalidCommand;
        return Command{ .stop = try allocator.dupe(u8, name) };
    } else if (std.mem.eql(u8, cmd_type, "purge")) {
        const name = lines.next() orelse return error.InvalidCommand;
        return Command{ .purge = try allocator.dupe(u8, name) };
    } else if (std.mem.eql(u8, cmd_type, "list")) {
        return Command.list;
    }

    return error.UnknownCommand;
}

fn spawnProcess(allocator: std.mem.Allocator, command: []const u8) !u32 {
    var args = std.ArrayList([]const u8).init(allocator);
    defer args.deinit();

    var cmd_iter = std.mem.splitSequence(u8, command, " ");
    while (cmd_iter.next()) |part| {
        try args.append(try allocator.dupe(u8, part));
    }
    try args.append(null);

    const pid = try posix.fork();

    if (pid == 0) {
        detachSession();
        const argv: [*:null]const ?[*:0]const u8 = @ptrCast(args.items.ptr);
        return posix.execveZ(args.items[0].?, argv, currentEnvironment());
    }

    return @intCast(pid);
}

fn handleCommand(allocator: std.mem.Allocator, cmd: Command, writer: anytype) !void {
    switch (cmd) {
        .start => |start_cmd| {
            const pid = try spawnProcess(allocator, start_cmd.command);
            registry.add(start_cmd.name, start_cmd.command, pid);
            try registry.save();
            try writer.print("started process {s} with pid {}\n", .{ start_cmd.name, pid });
        },
        .stop => |name| {
            if (registry.stopProcess(name)) {
                registry.remove(name);
                try registry.save();
            }
        },
        .purge => |name| {
            _ = registry.purge(name);
            try registry.save();
        },
        .list => {
            const processes = registry.getAll();
            for (processes) |p| {
                const status = switch (p.status) {
                    .running => "running",
                    .stopped => "stopped",
                };
                try writer.print("{s}  [{s}]  {s}\n", .{ p.name, status, p.command });
            }
        },
    }
}

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();

    const allocator = gpa.allocator();

    const stderr = std.io.getStdErr().writer();
    try stderr.print("zpmd daemon starting...\n", .{});

    try registry.load(allocator);

    try stderr.print("zpmd daemon started\n", .{});
}

const std = @import("std");
const builtin = @import("builtin");
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

pub const Process = struct {
    name: []const u8,
    command: []const u8,
    status: Status,
    pid: u32 = 0,
};

pub const Status = enum {
    running,
    stopped,
};

var processes = std.ArrayList(Process).init(std.heap.page_allocator);

const DATA_FILE = "zpm.json";

pub fn save() !void {
    const file = try std.fs.cwd().createFile(DATA_FILE, .{});
    defer file.close();

    try std.json.stringify(processes.items, .{}, file.writer());
}

pub fn load(allocator: std.mem.Allocator) !void {
    const file = std.fs.cwd().openFile(DATA_FILE, .{}) catch return;
    defer file.close();

    const data = try file.readToEndAlloc(allocator, 1024 * 1024);
    defer allocator.free(data);

    const parsed = try std.json.parseFromSlice(
        []Process,
        allocator,
        data,
        .{},
    );
    defer parsed.deinit();

    for (parsed.value) |p| {
        const name = try processes.allocator.dupe(u8, p.name);
        const command = try processes.allocator.dupe(u8, p.command);
        try processes.append(.{
            .name = name,
            .command = command,
            .status = p.status,
            .pid = p.pid,
        });
    }
}

pub fn add(name: []const u8, command: []const u8, pid: u32) void {
    processes.append(.{
        .name = name,
        .command = command,
        .status = .running,
        .pid = pid,
    }) catch {};

    save() catch {};
}

pub fn spawnProcess(allocator: std.mem.Allocator, command: []const u8) !u32 {
    const shell = try allocator.dupeZ(u8, "/bin/sh");
    defer allocator.free(shell);

    const shell_flag = try allocator.dupeZ(u8, "-c");
    defer allocator.free(shell_flag);

    const command_z = try allocator.dupeZ(u8, command);
    defer allocator.free(command_z);

    const pid = try posix.fork();

    if (pid == 0) {
        detachSession();

        const argv = [_:null]?[*:0]const u8{
            shell.ptr,
            shell_flag.ptr,
            command_z.ptr,
            null,
        };

        posix.execveZ(shell.ptr, &argv, currentEnvironment()) catch posix.exit(127);
        unreachable;
    }

    return @intCast(pid);
}

pub fn remove(name: []const u8) void {
    var i: usize = 0;

    while (i < processes.items.len) : (i += 1) {
        if (std.mem.eql(u8, processes.items[i].name, name)) {
            processes.items[i].status = .stopped;
            save() catch {};
            return;
        }
    }
}

pub fn purge(name: []const u8) bool {
    var i: usize = 0;

    while (i < processes.items.len) : (i += 1) {
        if (std.mem.eql(u8, processes.items[i].name, name)) {
            _ = processes.swapRemove(i);
            save() catch {};
            return true;
        }
    }

    return false;
}

pub fn getAll() []Process {
    return processes.items;
}

pub fn stopProcess(name: []const u8) bool {
    var i: usize = 0;

    while (i < processes.items.len) : (i += 1) {
        if (std.mem.eql(u8, processes.items[i].name, name)) {
            const pid = processes.items[i].pid;
            if (pid > 0) {
                const process_group: posix.pid_t = -@as(posix.pid_t, @intCast(pid));
                posix.kill(process_group, posix.SIG.TERM) catch |err| {
                    if (err != error.ProcessNotFound) {
                        return false;
                    }
                };
            }
            return true;
        }
    }

    return false;
}

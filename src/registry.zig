const std = @import("std");
const posix = std.posix;

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
                posix.kill(@intCast(pid), posix.SIG.TERM) catch |err| {
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

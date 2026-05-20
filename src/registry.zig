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

pub fn add(name: []const u8, command: []const u8, pid: u32) void {
    const allocator = processes.allocator;
    const name_copy = allocator.dupe(u8, name) catch return;
    const command_copy = allocator.dupe(u8, command) catch return;

    processes.append(.{
        .name = name_copy,
        .command = command_copy,
        .status = .running,
        .pid = pid,
    }) catch {
        allocator.free(name_copy);
        allocator.free(command_copy);
    };
}

pub fn spawnProcess(allocator: std.mem.Allocator, command: []const u8, pipe_fd: ?posix.fd_t) !u32 {
    const shell = try allocator.dupeZ(u8, "/bin/sh");
    defer allocator.free(shell);

    const shell_flag = try allocator.dupeZ(u8, "-c");
    defer allocator.free(shell_flag);

    const command_z = try allocator.dupeZ(u8, command);
    defer allocator.free(command_z);

    const pid = try posix.fork();

    if (pid == 0) {
        detachSession();

        if (pipe_fd) |w_fd| {
            try posix.dup2(w_fd, posix.STDOUT_FILENO);
            try posix.dup2(w_fd, posix.STDERR_FILENO);
            posix.close(w_fd);
        }

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
            return;
        }
    }
}

pub fn purge(name: []const u8) bool {
    var i: usize = 0;
    while (i < processes.items.len) : (i += 1) {
        if (std.mem.eql(u8, processes.items[i].name, name)) {
            const proc = processes.swapRemove(i);
            const allocator = processes.allocator;
            allocator.free(proc.name);
            allocator.free(proc.command);
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

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
    shutdown,
};

fn parseCommand(allocator: std.mem.Allocator, data: []const u8) !Command {
    var lines = std.mem.splitSequence(u8, data, "\n");
    const cmd_type = lines.next() orelse return error.InvalidCommand;

    if (std.mem.eql(u8, cmd_type, "stop_daemon")) {
        return Command.shutdown;
    } else if (std.mem.eql(u8, cmd_type, "start")) {
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

fn spawnProcess(allocator: std.mem.Allocator, name: []const u8, command: []const u8) !u32 {
    const log_dir = "logs";
    std.fs.cwd().makeDir(log_dir) catch |err| switch (err) {
        error.PathAlreadyExists => {},
        else => return err,
    };

    const log_filename = try std.fmt.allocPrint(allocator, "logs/{s}.log", .{name});
    defer allocator.free(log_filename);

    const log_file = try std.fs.cwd().createFile(log_filename, .{ .truncate = false });
    defer log_file.close();

    const stat = try log_file.stat();
    try log_file.seekTo(stat.size);

    const sh_path = "/bin/sh";
    const sh_flag = "-c";

    const sh_path_z = try allocator.dupeZ(u8, sh_path);
    defer allocator.free(sh_path_z);

    const sh_flag_z = try allocator.dupeZ(u8, sh_flag);
    defer allocator.free(sh_flag_z);

    const command_z = try allocator.dupeZ(u8, command);
    defer allocator.free(command_z);

    const pid = try posix.fork();

    if (pid == 0) {
        detachSession();

        try posix.dup2(log_file.handle, posix.STDOUT_FILENO);
        try posix.dup2(log_file.handle, posix.STDERR_FILENO);

        const argv = [_:null]?[*:0]const u8{
            sh_path_z.ptr,
            sh_flag_z.ptr,
            command_z.ptr,
            null,
        };

        _ = posix.execveZ(sh_path_z.ptr, &argv, currentEnvironment()) catch posix.exit(127);
        unreachable;
    }

    return @intCast(pid);
}

fn handleCommand(allocator: std.mem.Allocator, cmd: Command, writer: anytype, client_fd: posix.fd_t) !void {
    switch (cmd) {
        .shutdown => {
            _ = posix.write(client_fd, "success\n") catch {};
            posix.exit(0);
        },
        .start => |start_cmd| {
            const pid = try spawnProcess(allocator, start_cmd.name, start_cmd.command);
            registry.add(start_cmd.name, start_cmd.command, pid);
            // try registry.save();
            try writer.print("success\n{}\n", .{pid});
        },
        .stop => |name| {
            if (registry.stopProcess(name)) {
                registry.remove(name);
                // try registry.save();
                try writer.print("success\nstopped {s}\n", .{name});
            } else {
                try writer.print("error\nfailed to stop process\n", .{});
            }
        },
        .purge => |name| {
            if (registry.purge(name)) {
                // try registry.save();
                try writer.print("success\npurged {s}\n", .{name});
            } else {
                try writer.print("error\nprocess not found\n", .{});
            }
        },
        .list => {
            try writer.print("success\n", .{});
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

    const socket_path = "/tmp/zpm.sock";
    std.fs.deleteFileAbsolute(socket_path) catch |err| switch (err) {
        error.FileNotFound => {},
        else => return err,
    };

    //try registry.load(allocator);

    const server_fd = try posix.socket(posix.AF.UNIX, posix.SOCK.STREAM, 0);
    defer posix.close(server_fd);

    var addr = posix.sockaddr.un{
        .path = undefined,
    };
    std.mem.copyForwards(u8, &addr.path, socket_path);
    addr.path[socket_path.len] = 0;

    const sock_len = @as(posix.socklen_t, @intCast(@offsetOf(posix.sockaddr.un, "path") + socket_path.len + 1));
    try posix.bind(server_fd, @ptrCast(&addr), sock_len);
    try posix.listen(server_fd, 128);

    var client_buffer: [4096]u8 = undefined;

    while (true) {
        var client_addr: posix.sockaddr = undefined;
        var client_addr_len: posix.socklen_t = @sizeOf(posix.sockaddr);

        const client_fd = posix.accept(server_fd, &client_addr, &client_addr_len, 0) catch |err| {
            switch (err) {
                error.ConnectionAborted, error.ProcessFdQuotaExceeded, error.SystemFdQuotaExceeded => continue,
                else => return err,
            }
        };
        defer posix.close(client_fd);

        const bytes_read = posix.read(client_fd, &client_buffer) catch continue;
        if (bytes_read == 0) continue;

        const incoming_payload = client_buffer[0..bytes_read];

        var arena = std.heap.ArenaAllocator.init(allocator);
        defer arena.deinit();
        const arena_allocator = arena.allocator();

        const cmd = parseCommand(arena_allocator, incoming_payload) catch {
            _ = posix.write(client_fd, "error\nfailed to parse command payload\n") catch {};
            continue;
        };

        var response_list = std.ArrayList(u8).init(arena_allocator);
        const writer = response_list.writer();

        handleCommand(arena_allocator, cmd, writer, client_fd) catch {
            _ = posix.write(client_fd, "error\ninternal runtime failure execution\n") catch {};
            continue;
        };

        _ = posix.write(client_fd, response_list.items) catch {};
    }
}

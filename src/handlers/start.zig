const std = @import("std");
const registry = @import("../registry.zig");

/// Extract script from package.json
/// Returns allocated script command or null if not found
fn getPackageJsonScript(allocator: std.mem.Allocator, script_name: []const u8) ?[]const u8 {
    const cwd = std.fs.cwd();
    const file = cwd.openFile("package.json", .{}) catch return null;
    defer file.close();

    const content = file.readToEndAlloc(allocator, 1024 * 1024) catch return null;
    defer allocator.free(content);

    var parsed = std.json.parseFromSlice(std.json.Value, allocator, content, .{}) catch return null;
    defer parsed.deinit();

    const root = parsed.value.object;
    const scripts = root.get("scripts") orelse return null;

    if (scripts != .object) return null;

    const script = scripts.object.get(script_name) orelse return null;

    if (script != .string) return null;

    return allocator.dupe(u8, script.string) catch return null;
}

pub fn handleStart(args: *std.process.ArgIterator) void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

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

    // Parse optional --script and command arguments
    var script_name: []const u8 = "start";
    var command_opt: ?[]const u8 = null;

    while (args.next()) |arg| {
        if (std.mem.eql(u8, arg, "--script")) {
            script_name = args.next() orelse {
                std.debug.print("missing script name after --script\n", .{});
                return;
            };
        } else {
            command_opt = arg;
        }
    }

    // Determine the command to run
    const command = if (command_opt) |cmd| cmd else blk: {
        // Try to get from package.json
        if (getPackageJsonScript(allocator, script_name)) |script_cmd| {
            std.debug.print("using script from package.json: {s}\n", .{script_name});
            break :blk script_cmd;
        } else {
            std.debug.print("error: no command provided and no package.json script found\n", .{});
            std.debug.print("usage: zpm start --name <name> [--script <script>] [command]\n", .{});
            return;
        }
    };
    defer if (command_opt == null) allocator.free(command);

    const pid = registry.spawnProcess(allocator, command) catch |err| {
        std.debug.print("failed to start process: {}\n", .{err});
        return;
    };

    registry.add(name, command, pid);

    std.debug.print("process started\n", .{});
    std.debug.print("name: {s}\n", .{name});
    std.debug.print("command: {s}\n", .{command});
    std.debug.print("pid: {}\n", .{pid});
}

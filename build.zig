const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const exe = b.addExecutable(.{
        .name = "zpm",
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });

    b.installArtifact(exe);

    const daemon = b.addExecutable(.{
        .name = "zpmd",
        .root_source_file = b.path("src/daemon.zig"),
        .target = target,
        .optimize = optimize,
    });

    b.installArtifact(daemon);
}

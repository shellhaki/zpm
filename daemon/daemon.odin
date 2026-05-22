package main

import "../models"
import "../registry"
import "core:fmt"
import "core:net"
import "core:os"
import "core:strings"
import "core:sys/unix"

// POSIX file open flags
O_WRONLY :: 1

main :: proc() {
	endpoint := net.Endpoint{
		address = net.IP4_Loopback,
		port    = 5890,
	}

	listener, listen_err := net.listen_tcp(endpoint)
	if listen_err != nil {
		fmt.eprintln("Failed to start daemon listener:", listen_err)
		return
	}
	defer net.close(listener)

	fmt.println("ZPM Daemon successfully started on port 5890...")

	for {
		client, _, accept_err := net.accept_tcp(listener)
		if accept_err != nil {
			continue
		}

		handle_client(client)
	}
}

handle_client :: proc(sock: net.TCP_Socket) {
	defer net.close(sock)

	buf: [1024]u8
	n, recv_err := net.recv_tcp(sock, buf[:])
	if recv_err != nil || n == 0 {
		return
	}

	payload := string(buf[:n])
	parts := strings.split(payload, "|")
	defer delete(parts)

	if len(parts) < 3 {
		net.send_tcp(sock, transmute([]u8)"Error: Malformed protocol message")
		return
	}

	action := parts[0]
	process_name := parts[1]
	command_string := parts[2]

	if action == "start" {
		fmt.printf("Received request to start [%s]: %s\n", process_name, command_string)

		cmd_args := strings.split(command_string, " ")
		defer delete(cmd_args)

		// Open /dev/null for output redirection
		null_fd := unix.sys_open(cstring("/dev/null"), O_WRONLY, 0)
		if null_fd < 0 {
			net.send_tcp(sock, transmute([]u8)"Error: Failed to open null device")
			return
		}
		defer unix.sys_close(null_fd)

		// Fork child process
		pid := unix.sys_fork()
		if pid < 0 {
			net.send_tcp(sock, transmute([]u8)"Error: Failed to fork process")
			return
		}

		if pid == 0 {
			// Child process: redirect output and execute command
			unix.sys_dup2(null_fd, 0)  // stdin
			unix.sys_dup2(null_fd, 1)  // stdout
			unix.sys_dup2(null_fd, 2)  // stderr

			// Convert command args to C-compatible format for execve
			argv := make([dynamic]cstring)
			defer delete(argv)
			for arg in cmd_args {
				append(&argv, cstring(raw_data(arg)))
			}
			append(&argv, nil)

			// Execute the command via execve syscall
			// sys_execve is a raw syscall, returns i32
			unix.sys_execve(cstring(raw_data(cmd_args[0])), raw_data(argv[:]), nil)
			
			// If execve returns, it failed - exit child
			os.exit(1)
		}

		// Parent process: register and respond
		pid_u32 := u32(pid)
		registry.Add(process_name, command_string, pid_u32)

		success_msg := fmt.aprintf("Success: Process [%s] started with PID %d", process_name, pid_u32)
		defer delete(success_msg)
		net.send_tcp(sock, transmute([]u8)success_msg)
	} else {
		net.send_tcp(sock, transmute([]u8)"Error: Unknown action command")
	}
}

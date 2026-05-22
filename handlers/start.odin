package handlers
import "../src"
import "core:net"
import "core:strings"
import "../registry"
import "core:fmt"

Handlestart :: proc (args: []string) -> bool {
    if len(args) < 4{
        fmt.println("invalid usage, eg: zpm start <command> --name <process name>")
        return false
    }
    name_index := -1
    for i in 0..<len(args){
        if args[i] == "--name"{
            name_index = i
            break
        }
       
    }
    if name_index == -1{
        fmt.println("name flag missing eg: --name <process name>")
        return false
    }
    if name_index + 1 >= len(args){
        fmt.println("process name missing, eg: --name <process name>")
        return false
    }
    command_parts := args[2:name_index]
    if len(command_parts) == 0{
        fmt.println("command missing, eg: zpm start <command> --name <process name>")
        return false
    }
    command := strings.join(command_parts, " ")
    defer delete(command)
    name := args[name_index + 1]
    sock,err := net.dial_tcp("127.0.0.1:5890")
    if err != nil{
        fmt.println("an error occured while dialing tcp")
        return false
    }
    defer net.close(sock)
    message := strings.concatenate({"start|",name,"|",command})
    defer  delete(message)
    _,err = net.send_tcp(sock,transmute([]u8)message)
    if err != nil{
        fmt.println("failed to send full command to daemon")
        return false
    }
    buff: [1024]u8
    n,errr := net.recv_tcp(sock,buff[:])   
    if errr != nil {
        fmt.println("failed to receive full respoinse from daemon")
    }

    fmt.println(string(buff[:n]))
    return true
}
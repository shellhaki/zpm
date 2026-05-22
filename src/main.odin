package main
import "core:fmt"
import "core:os"
import "../setup"

main :: proc (){
    setup.Init()
    if len(os.args) < 2 {
        fmt.println("arguement missing")
        return 
    }
    if action := os.args[1]; action == "start"{

    } else if action == "stop"{
        fmt.println("process stopped")
    } else if action == "list"{
        fmt.println("listing...")
    } else if action == "purge" {
        fmt.println("process purged")
    }
}

/* 
./zpm - 0
start - 1
command - 2

*/
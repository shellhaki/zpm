package handlers
import "../src"
import "../registry"
import "core:fmt"
HandleStart :: proc(args: []string) -> bool {
    if length := len(args); length < 5 {
        fmt.println("Incomplete args, required, eg: <zpm start 'node index.js' --name test>")
        return false
    }
    command := args[2]
    if command == "" {
        fmt.println("command arguement missing, eg: <zpm start 'node index.js' --name test>")
        return false
    }
    nameFlag := args[3]
    if nameFlag == ""{
        fmt.println("flag arguement missing, eg: <zpm start 'node index.js' --name test>")
        return false
    }else if nameFlag != "--name"{
        fmt.println("incorrect flag, use --name, eg: <zpm start 'node index.js' --name test>")
        return false
    }
    name := args[4]
    if name == ""{
        fmt.println("name argument missing, eg: <zpm start 'node index.js' --name test>")
        return false
    }
    ok := registry.Add(name,command)
    if !ok{
        fmt.println("an error occured while adding process")
        return false
    }
    return true

}

/* zpm start "./index" --name haki */
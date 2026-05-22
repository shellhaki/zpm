package registry
import "../models"
import "../setup"
Add :: proc(name: string, command: string) -> bool{
    if name == "" || command == ""{
        return false
    }
    append(&setup.Processes, models.Process{
        name = name,
        command = command,

    })
    return true
}
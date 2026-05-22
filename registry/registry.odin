package registry
import "../models"
import "../setup"
Add :: proc(name: string, command: string, pid: u32) -> bool{
    if name == "" || command == "" || pid == 0{
        return false
    }
    append(&setup.Processes, models.Process{
        name = name,
        command = command,
        pid = pid

    })
    return true
}
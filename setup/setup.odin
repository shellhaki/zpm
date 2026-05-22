package setup
import "../models"

// processes variable definition
Processes: [dynamic]models.Process

Init :: proc(){
    Processes = make([dynamic]models.Process,context.allocator)
}
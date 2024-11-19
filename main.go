package main

import (
    "fmt"
    "time"

    "gobot.io/x/gobot"
    "gobot.io/x/gobot/platforms/dji/tello"
)

func main() {
    drone := tello.NewDriver("8888") // Telloドローンのデフォルトポート

    work := func() {
        drone.TakeOff()

        gobot.After(10*time.Second, func() {
            drone.Land()
        })
    }

    robot := gobot.NewRobot(
        []gobot.Connection{},
        []gobot.Device{drone},
        work,
    )

    err := robot.Start()
    if err != nil {
        fmt.Println("Error starting robot:", err)
    }
}

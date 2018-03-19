package main

import (
	"bytes"
	"net/http"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"time"
	"strings"
	"os"
    "os/exec"
	"runtime"
	"bufio"
)

const (
	CreateNewMazeAPI			= "https://ponychallenge.trustpilot.com/pony-challenge/maze"
	GetMazeCurrentStateAPI		= "https://ponychallenge.trustpilot.com/pony-challenge/maze/"
	MakeNextMoveAPI				= "https://ponychallenge.trustpilot.com/pony-challenge/maze/"
	GetVisualOfCurrentStateAPI	= "https://ponychallenge.trustpilot.com/pony-challenge/maze/{maze-id}/print"
)

type NewMaze struct {
	MazeWidth int `json:"maze-width"`
	MazeHeight int `json:"maze-height"`
	MazePlayerName string `json:"maze-player-name"`
	Difficulty int `json:"difficulty"`
}

type Maze struct {
	Pony       []int      `json:"pony"`
	Domokun    []int      `json:"domokun"`
	EndPoint   []int      `json:"end-point"`
	Size       []int      `json:"size"`
	Difficulty int        `json:"difficulty"`
	Data       [][]string `json:"data"`
	MazeID     string     `json:"maze_id"`
	GameState  struct {
		State       string `json:"state"`
		StateResult string `json:"state-result"`
	} `json:"game-state"`
}

type Move struct {
	State string `json:"state"`
	StateResult string `json:"state-result"`
}

var maze Maze
var goal_route []int
var goal bool

func InitiateNewMaze(width int, height int, name string, difficulty int) (error) {
	//Create New Maze
	newMaze := NewMaze{
		MazeWidth: width,
		MazeHeight: height,
		MazePlayerName: name,
		Difficulty: difficulty,
	}
	info := fmt.Sprintf("Creating new maze. Width:%v Height:%v Player:%s Difficulty:%v", width, height, name, difficulty)
	fmt.Println(info)
	jsonMaze, _ := json.Marshal(newMaze)

	//Post request to create a new maze
	res, err := http.Post(CreateNewMazeAPI,"application/json",bytes.NewBuffer(jsonMaze))
	if err != nil {
		return err
	}

	//Read response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	
	//Convert json to struct
	err = json.Unmarshal(body,&maze)
	if err != nil {
		return err
	}
	return nil
}

func GetMazeCurrentState(maze_id string) {
	fmt.Println("Getting current state of maze ", maze_id)
	//Get request to get maze current state
	res, err := http.Get(GetMazeCurrentStateAPI + maze_id)
	if err != nil {
		return
	}

	//Read response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	//Convert json to struct
	err = json.Unmarshal(body,&maze)
	if err != nil {
		return
	}
	fmt.Println("Getting current state of maze ", maze_id, " - successful")
}

func GetRouteToEndPoint() {
	fmt.Println("Calculating route to end point...")
	pony := maze.Pony[0]
	var steps []int
	initial_steps := append(steps, pony)
	runMaze(pony,initial_steps)
	fmt.Println("Done! Total move to end point are ", len(goal_route), " step(s)!")
}

func runMaze(point int, steps []int) ([]int) {
	ori_steps := steps
	width := maze.Size[0]
	dimension := maze.Size[0] * maze.Size[1]
	coordinate := maze.Data[point]
	//if out of dimension, "build" wall to indicate not walkable
	var coordinate_east []string
	if point+1 < dimension {
		coordinate_east = maze.Data[point+1]
	} else { 
		coordinate_east = []string{"west"}
	}
	var coordinate_south []string
	if point+width < dimension {
		coordinate_south = maze.Data[point+width]
	} else {
		coordinate_south = []string{"north"}
	}

	if point == maze.EndPoint[0] {
		//goal!
		goal_route = steps
		goal = true
		return nil
	}

	if !hasWall(coordinate,"west") && !goal {
		//move to west
		west := point-1
		if !turningBack(steps, west) {
			split_west := append(steps, west)
			steps = append(steps,runMaze(west, split_west)...)
		}
	}

	if !hasWall(coordinate,"north") && !goal {
		//move to north
		north := point-width
		if !turningBack(steps, north) {
			split_north := append(steps, north)
			steps = append(steps,runMaze(north, split_north)...)
		}
	}

	if !hasWall(coordinate_east,"west") && !goal {
		//move to east
		east := point+1
		if !turningBack(steps, east) {
			split_east := append(steps, east)
			steps = append(steps,runMaze(east, split_east)...)
		}
	}

	if !hasWall(coordinate_south,"north") && !goal {
		//move to south
		south := point+width
		if !turningBack(steps, south) {
			split_south := append(steps, south)
			steps = append(steps,runMaze(south, split_south)...)
		}
	}

	if len(ori_steps) == len(steps) {
		//dead end
		return nil
	}

	return steps
}

func hasWall(coordinate []string, direction string) (bool) {
	for _, wall := range coordinate {
		if wall == direction {
		   return true
		}
	 }
	 return false
}

func turningBack(steps []int, next_point int) (bool) {
	for _, point := range steps {
		if point == next_point {
		   return true
		}
	 }
	 return false
}

func StartWalking() {
	steps := goal_route
	reachGoal := false
	i := 1
	for !reachGoal {
		if i > len(steps) {
			break
		}

		previous := steps[i-1]
		current := steps[i]
		move := current - previous
		direction := "stay"
		if move == maze.Size[0]*-1 {
			direction = "north"
		} else if move == maze.Size[0] {
			direction = "south"
		} else if move == -1 {
			direction = "west"
		} else if move == 1 {
			direction = "east"
		}

		res, err, moveState := PostNextMove(direction)
		VisualizeCurrentState(moveState,i,len(steps))
		if moveState.State == "won" || moveState.State == "over" {
			break
		}

		if direction != "stay" && err == nil && res == 200 {
			i++
		}
		//Delay 0.5 second before executing next move
		time.Sleep(500 * time.Millisecond)
	}
}

func VisualizeCurrentState(move Move, current_step_index int, total_steps int) {
	//Get request to get visual of maze current state
	url := strings.Replace(GetVisualOfCurrentStateAPI,"{maze-id}",maze.MazeID,-1)
	res, err := http.Get(url)
	if err != nil {
		return
	}

	//Read response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	ClearTerminal()
	fmt.Println("Maze id: ", maze.MazeID)
	fmt.Println(string(body))
	info := fmt.Sprintf("%s. Step %v of %v", move.StateResult, current_step_index, total_steps-1)
	fmt.Println(info)
}

func PostNextMove(direction string) (int, error, Move) {
	var jsonVal = []byte(fmt.Sprintf(`{"direction": "%s"}`,direction))

	//Post request to make next move
	res, err := http.Post(MakeNextMoveAPI+maze.MazeID,"application/json",bytes.NewBuffer(jsonVal))
	if err != nil {
		return 500, err, Move{}
	}

	//Read response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 500, err, Move{}
	}

	//Convert json to struct
	var move Move
	err = json.Unmarshal(body,&move)
	if err != nil {
		return 500, err, Move{}
	}



	return res.StatusCode, nil, move
}

func ClearTerminal() {
	var c *exec.Cmd
	var doClear = true
	
	switch runtime.GOOS {
	case "darwin":
	case "linux":
		c = exec.Command("clear")
	case "windows":
		c = exec.Command("cmd", "/c", "cls")
	default:
		fmt.Println("Clear function not supported on current OS\n")
		doClear = false
	}
	if doClear {
		c.Stdout = os.Stdout
		c.Run()
	}
}

func main() {
	//fixed input for now
	err := InitiateNewMaze(15,15,"Fluttershy",10)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	GetMazeCurrentState(maze.MazeID)
	GetRouteToEndPoint()
	StartWalking()
	fmt.Print("Press 'Enter' to close...")
	bufio.NewReader(os.Stdin).ReadBytes('\n') 
}
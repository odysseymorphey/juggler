package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

type Ball struct {
	ID       int
	State    string
	Duration int
	Elapsed  int
}

var (
	ballsLock   sync.Mutex
	balls       []*Ball
	fallenBalls chan int
	wg          sync.WaitGroup
)

func throwBall(ctx context.Context, id int) {
	defer wg.Done()

	duration := rand.Intn(6) + 5 // случайное время 5-10 сек

	ballsLock.Lock()
	balls[id].State = "flying"
	balls[id].Duration = duration
	balls[id].Elapsed = 0
	ballsLock.Unlock()

	for i := 1; i <= duration; i++ {
		select {
		case <-ctx.Done():
			ballsLock.Lock()
			balls[id].State = "in_hands"
			ballsLock.Unlock()
			return
		case <-time.After(time.Second):
			ballsLock.Lock()
			balls[id].Elapsed = i
			fmt.Printf("Ball %d: %d/%d\n", id, i, duration)
			ballsLock.Unlock()
		}
	}

	ballsLock.Lock()
	balls[id].State = "in_hands"
	ballsLock.Unlock()

	select {
	case fallenBalls <- id:
	case <-ctx.Done():
	}
}

func printState() {
	ballsLock.Lock()
	defer ballsLock.Unlock()

	flying := 0
	inHands := 0
	for _, ball := range balls {
		if ball.State == "flying" {
			flying++
		} else {
			inHands++
		}
	}

	fmt.Println("\n=== Balls condition ===")
	fmt.Printf("In flying: %d, In hands: %d\n", flying, inHands)
	for _, ball := range balls {
		state := ball.State
		if state == "flying" {
			fmt.Printf("Ball %d: Flying (%d/%d сек)\n", ball.ID, ball.Elapsed, ball.Duration)
		} else {
			fmt.Printf("Ball %d: In hands\n", ball.ID)
		}
	}
	fmt.Println("======================")
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Использование: ./juggler <N> <T>")
		return
	}

	N, err := strconv.Atoi(os.Args[1])
	if err != nil || N <= 0 {
		fmt.Println("N must be a positive integer")
		return
	}

	T, err := strconv.Atoi(os.Args[2])
	if err != nil || T <= 0 {
		fmt.Println("T must be a positive integer")
		return
	}

	rand.Seed(time.Now().UnixNano())
	balls = make([]*Ball, N)
	for i := 0; i < N; i++ {
		balls[i] = &Ball{ID: i, State: "in_hands"}
	}

	fallenBalls = make(chan int, N)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timer := time.NewTimer(time.Duration(T) * time.Minute)
	ticker := time.NewTicker(time.Second)
	defer timer.Stop()
	defer ticker.Stop()

	initialThrown := 0
	timeExpired := false

	fmt.Printf("Start juggling: %d ball, %d minutes\n", N, T)
	printState()

	for {
		if timeExpired && initialThrown >= N {
			break
		}

		select {
		case <-timer.C:
			timeExpired = true
			cancel()
			ticker.Stop()
			fmt.Println("Juggling time is up. Waiting for the balls to fall...")
		case id := <-fallenBalls:
			if !timeExpired {
				printState()
				wg.Add(1)
				go throwBall(ctx, id)
			}
		case <-ticker.C:
			if !timeExpired && initialThrown < N {
				printState()
				wg.Add(1)
				go throwBall(ctx, initialThrown)
				initialThrown++
			}
		}
	}

	wg.Wait()
	fmt.Println("All balls is fallen. Finishing work")
}
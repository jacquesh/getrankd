package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"getrankd/api"
)

type ImportPlayerResult struct {
	NewRank int
	Score   int
	Name    string
}

type ImportMatchResult struct {
	Players   []ImportPlayerResult
	GameName  string
	Timestamp time.Time
}

func importFromCsv(filename string) {
	startTime := time.Now()
	fmt.Println("Import from CSV")
	fileHandle, err := os.Open(filename)
	defer fileHandle.Close()
	if err != nil {
		panic(err)
	}

	matches := make([]ImportMatchResult, 0)
	reader := bufio.NewReader(fileHandle)
	for {
		lineContents, readLineErr := reader.ReadString('\n')
		if (readLineErr != nil) && (readLineErr != io.EOF) {
			panic(readLineErr)
		}
		lineFields := strings.Split(strings.TrimSpace(lineContents), ",")

		playerCount := (len(lineFields) - 2) / 3
		if playerCount <= 0 {
			break
		}

		timestampStr := lineFields[0]
		gameName := strings.Trim(lineFields[1], "\"")

		timestamp, _ := time.Parse("2006-01-02 15:04:05", timestampStr[1:len(timestampStr)-1])

		players := make([]ImportPlayerResult, playerCount)
		lineFieldIndex := 2
		for i := 0; i < playerCount; i++ {
			playerRankStr := lineFields[lineFieldIndex+0]
			playerScoreStr := lineFields[lineFieldIndex+1]
			playerName := strings.Trim(lineFields[lineFieldIndex+2], "\"")

			playerRank, _ := strconv.Atoi(playerRankStr)
			playerScore, _ := strconv.Atoi(playerScoreStr)
			newPlayer := ImportPlayerResult{
				NewRank: playerRank,
				Score:   playerScore,
				Name:    playerName,
			}
			players[i] = newPlayer
			lineFieldIndex += 3
		}

		newMatch := ImportMatchResult{
			Players:   players,
			GameName:  gameName,
			Timestamp: timestamp,
		}
		fmt.Println(newMatch)
		matches = append(matches, newMatch)

		if readLineErr != nil {
			break
		}
	}

	gameIdMap := make(map[string]int64)
	playerIdMap := make(map[string]int64)
	for matchIndex := 0; matchIndex < len(matches); matchIndex++ {
		match := matches[len(matches)-matchIndex-1] // NOTE: Matches are in reverse chronological order in the file
		gameId, hasGameId := gameIdMap[match.GameName]
		if !hasGameId {
			gameId = api.PersistNewGame(match.GameName)
			gameIdMap[match.GameName] = gameId
		}

		playerResults := make([]api.PlayerMatchResult, 0)
		for playerPosition, player := range match.Players {
			playerId, hasPlayerId := playerIdMap[player.Name]
			if !hasPlayerId {
				playerId = api.PersistNewPlayer(player.Name)
				playerIdMap[player.Name] = playerId
			}

			newPlayer := api.PlayerMatchResult{
				PlayerId: playerId,
				// Score
				Position: playerPosition,
			}
			playerResults = append(playerResults, newPlayer)
		}

		api.PersistNewMatch(gameId, match.Timestamp, playerResults)
		fmt.Printf("Persisted match %d/%d\n", matchIndex+1, len(matches))
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Import completed in %s\n", elapsedTime)
}

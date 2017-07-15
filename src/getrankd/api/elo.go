package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

/*
General ranking systems:
- http://www.lifewithalacrity.com/2006/01/ranking_systems.html

ELO:
- https://gamedev.stackexchange.com/questions/55441/player-ranking-using-elo-with-more-than-two-players
- http://elo-norsak.rhcloud.com/3.php
- https://github.com/FigBug/Multiplayer-ELO

TrueSkill:
- https://www.microsoft.com/en-us/research/project/trueskill-ranking-system/
*/

func GetRankHistory(write http.ResponseWriter, req *http.Request) {
	result := getRatingHistory()
	write.Header().Set("Content-Type", "application/json")
	json.NewEncoder(write).Encode(result)
}

func AddGame(write http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		panic(err)
	}

	if req.Method != "POST" {
		write.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name, hasName := req.PostForm["gameName"]
	if !hasName {
		write.WriteHeader(http.StatusBadRequest)
		return
	}

	PersistNewGame(name[0])
	http.Redirect(write, req, "/", http.StatusFound) // TODO: What status should we be using?
}

func AddPlayer(write http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		panic(err)
	}

	if req.Method != "POST" {
		write.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name, hasName := req.PostForm["playerName"]
	if !hasName {
		write.WriteHeader(http.StatusBadRequest)
		return
	}

	PersistNewPlayer(name[0])
	http.Redirect(write, req, "/", http.StatusFound) // TODO: What status should we be using?
}

func AddMatch(write http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		panic(err)
	}

	if req.Method != "POST" {
		write.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	gameId, hasGameId := req.PostForm["gameId"]
	playerIds, hasPlayers := req.PostForm["playerId"]
	positions, hasPositions := req.PostForm["playerPosition"]
	if !hasGameId || !hasPlayers || !hasPositions || (len(playerIds) != len(positions)) {
		write.WriteHeader(http.StatusBadRequest)
		return
	}

	parsedGameId, err := strconv.ParseInt(gameId[0], 10, 64)
	if err != nil {
		panic(err)
	}

	results := make([]PlayerMatchResult, len(playerIds))
	for i := 0; i < len(playerIds); i++ {
		id, parseErr := strconv.ParseInt(playerIds[i], 10, 64)
		if parseErr != nil {
			write.WriteHeader(http.StatusBadRequest)
			return
		}

		position, parseErr := strconv.Atoi(positions[i])
		if parseErr != nil {
			write.WriteHeader(http.StatusBadRequest)
			return
		}

		results[i] = PlayerMatchResult{
			PlayerId: id,
			Position: position,
		}
	}

	PersistNewMatch(parsedGameId, time.Now(), results)
	http.Redirect(write, req, "/", http.StatusFound) // TODO: What status should we be using?
}

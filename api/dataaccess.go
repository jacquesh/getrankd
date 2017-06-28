package api

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type GameData struct {
	Id   int64
	Name string
}

type MatchData struct {
	Name      string
	Timestamp time.Time
}

type PlayerData struct {
	Id    int64
	Name  string
	Score float64
}

type HistoryEntry struct {
	MatchId   int64
	Timestamp time.Time
	Rating    float64
}

type PlayerHistory struct {
	Name    string
	Ratings []HistoryEntry
}

type PlayerMatchResult struct {
	PlayerId int64
	Position int
}

var db *sql.DB

func GetRecentMatches() []MatchData {
	cmd := `
		SELECT
			Game.Name,
			Match.Timestamp
		FROM
			Match
			INNER JOIN Game
			ON Game.Id = Match.GameId
		ORDER BY
			Timestamp DESC
		LIMIT 10`
	rows, err := db.Query(cmd)
	if err != nil {
		panic(err)
	}

	result := make([]MatchData, 0)
	var name string
	var timestamp time.Time

	for rows.Next() {
		err = rows.Scan(&name, &timestamp)

		newValue := MatchData{
			Name:      name,
			Timestamp: timestamp,
		}
		result = append(result, newValue)
	}
	rows.Close()
	return result
}

func getRatingHistory() []PlayerHistory {
	playerMap := make(map[int64]string)
	rows, err := db.Query("SELECT Id, Name FROM Player")
	if err != nil {
		panic(err)
	}

	var id int64
	var name string
	for rows.Next() {
		rows.Scan(&id, &name)
		playerMap[id] = name
	}
	rows.Close()

	rows, err = db.Query(`
		SELECT
			Match.Id,
			PlayerMatch.PlayerId,
			PlayerMatch.NewElo,
			Match.Timestamp
		FROM
			Match INNER JOIN PlayerMatch
			ON PlayerMatch.MatchId = Match.Id
		ORDER BY
			Match.Timestamp ASC`)
	if err != nil {
		panic(err)
	}

	playerHistories := make(map[int64][]HistoryEntry)
	var matchId int64
	var playerId int64
	var playerElo float64
	var timestamp time.Time
	for rows.Next() {
		rows.Scan(&matchId, &playerId, &playerElo, &timestamp)

		newHistoryEntry := HistoryEntry{
			MatchId:   matchId,
			Timestamp: timestamp,
			Rating:    playerElo,
		}

		history, ok := playerHistories[playerId]
		if ok {
			playerHistories[playerId] = append(history, newHistoryEntry)
		} else {
			playerHistories[playerId] = []HistoryEntry{newHistoryEntry}
		}
	}
	rows.Close()

	result := make([]PlayerHistory, 0)
	for playerId, historyEntries := range playerHistories {
		history := PlayerHistory{
			Name:    playerMap[playerId],
			Ratings: historyEntries,
		}
		result = append(result, history)
	}
	return result
}

func GetAllGameData() []GameData {
	rows, err := db.Query("SELECT Id, Name FROM Game")
	if err != nil {
		panic(err)
	}

	result := make([]GameData, 0)
	var id int64
	var name string
	for rows.Next() {
		rows.Scan(&id, &name)
		value := GameData{
			Id:   id,
			Name: name,
		}
		result = append(result, value)
	}
	return result
}

func GetAllPlayerData() []PlayerData {
	rows, err := db.Query("SELECT Id, Name, Elo FROM Player ORDER BY Elo DESC")
	if err != nil {
		panic(err)
	}

	result := make([]PlayerData, 0)
	var id int64
	var name string
	var score float64
	for rows.Next() {
		rows.Scan(&id, &name, &score)
		value := PlayerData{
			Id:    id,
			Name:  name,
			Score: score,
		}
		result = append(result, value)
	}
	rows.Close()
	return result
}

func getRatingMap() map[int64]float64 {
	rows, err := db.Query("SELECT Id, Elo FROM Player")
	if err != nil {
		panic(err)
	}

	result := make(map[int64]float64)
	var id int64
	var score float64
	for rows.Next() {
		rows.Scan(&id, &score)
		result[id] = score
	}
	rows.Close()
	return result
}

func Initialize() {
	os.Remove("./ratings.db")
	var err error
	db, err = sql.Open("sqlite3", "./ratings.db")
	if err != nil {
		panic(err)
	}

	if _, err = os.Stat("./ratings.db"); os.IsNotExist(err) {
		_, err = db.Exec("PRAGMA synchronous = NORMAL")
		_, err = db.Exec("PRAGMA journal_mode = WAL")
		_, err = db.Exec(`
			CREATE TABLE Game (
				Id      INTEGER PRIMARY KEY NOT NULL,
				Name    TEXT NOT NULL
			);
			CREATE TABLE Player (
				Id      INTEGER PRIMARY KEY NOT NULL,
				Name    TEXT NOT NULL,
				Elo     FLOAT NOT NULL DEFAULT(1500)
			);
			CREATE TABLE Match (
				Id          INTEGER PRIMARY KEY NOT NULL,
				GameId      INT NOT NULL,
				Timestamp   DATETIME NOT NULL
			);
			CREATE TABLE PlayerMatch (
				PlayerId    INT NOT NULL,
				MatchId     INT NOT NULL,
				EloDelta    FLOAT NOT NULL,
				NewElo      FLOAT NOT NULL
			);`)
		if err != nil {
			panic(err)
		}
	}
}

func PersistNewGame(name string) int64 {
	fmt.Printf("Persist new game #%s#\n", name)
	result, err := db.Exec("INSERT INTO Game(Name) VALUES(?1)", name)
	if err != nil {
		panic(err)
	}

	newGameId, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}
	return newGameId
}

func PersistNewPlayer(name string) int64 {
	result, err := db.Exec("INSERT INTO Player(Name) VALUES(?1)", name)
	if err != nil {
		fmt.Printf("Error: Unable to add player %s: %s\n", name, err)
		panic(err)
	}

	newPlayerId, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}
	return newPlayerId
}

func PersistNewMatch(gameId int64, timestamp time.Time, results []PlayerMatchResult) {
	eloChanges := make(map[int64]float64)
	for _, result := range results {
		eloChanges[result.PlayerId] = 0
	}

	currentElos := getRatingMap()
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			var iScore, jScore float64
			if results[i].Position < results[j].Position {
				iScore = 1.0
				jScore = 0.0
			} else if results[i].Position > results[j].Position {
				iScore = 0.0
				jScore = 1.0
			} else {
				iScore = 0.5
				jScore = 0.5
			}

			iPlayer := results[i].PlayerId
			jPlayer := results[j].PlayerId
			iExpected := 1.0 / (1 + math.Pow(10, (currentElos[jPlayer]-currentElos[iPlayer])/400))
			jExpected := 1.0 / (1 + math.Pow(10, (currentElos[iPlayer]-currentElos[jPlayer])/400))

			k := 32.0 / float64(len(results)-1.0)
			eloChanges[iPlayer] += k * (iScore - iExpected)
			eloChanges[jPlayer] += k * (jScore - jExpected)
		}
	}

	result, err := db.Exec("INSERT INTO Match(GameId, Timestamp) VALUES(?1, ?2)", gameId, timestamp)
	if err != nil {
		panic(err)
	}
	matchId, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}

	for playerId, scoreDelta := range eloChanges {
		newScore := currentElos[playerId] + scoreDelta
		_, err = db.Exec("UPDATE Player SET Elo = Elo + ?1 WHERE Id = ?2", scoreDelta, playerId)
		if err != nil {
			panic(err)
		}

		_, err = db.Exec("INSERT INTO PlayerMatch(PlayerId, MatchId, EloDelta, NewElo) VALUES(?1, ?2, ?3, ?4)",
			playerId, matchId, scoreDelta, newScore)
		if err != nil {
			panic(err)
		}
	}
}

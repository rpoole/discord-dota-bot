package main

import (
	"time"
	"log"
	"github.com/mxk/go-sqlite/sqlite3"
	"fmt"
	"strings"
	"math"
)

func getPlayerStreak(db *sqlite3.Conn, player PlayerData) int {
	sql := "SELECT streak FROM players where account_id = " + player.accountId
	row, err := db.Query(sql)
	var streak int
	err = row.Scan(&streak)
	if err != nil {
		log.Println("Failed to db.Query:", err)
	}
	log.Println(streak)

	return streak
}

func getNextDay(db *sqlite3.Conn) time.Time {
	sql := "SELECT strftime('%s', value) FROM settings where setting ='next_day';"
	row, err := db.Query(sql)
	var nextDay time.Time
	err = row.Scan(&nextDay)
	if err != nil {
		log.Println("Failed to db.Query:", err)
	}
	log.Println(nextDay)

	return nextDay
}

func getNextWeek(db *sqlite3.Conn) time.Time {
	sql := "SELECT strftime('%s', value) FROM settings where setting ='next_week';"
	row, err := db.Query(sql)
	var nextWeek time.Time
	err = row.Scan(&nextWeek)
	if err != nil {
		log.Println("Failed to db.Query:", err)
	}
	log.Println(nextWeek)

	return nextWeek
}

func getNextMonth(db *sqlite3.Conn) time.Time {
	sql := "SELECT strftime('%s', value) FROM settings where setting ='next_month';"
	row, err := db.Query(sql)
	var nextMonth time.Time
	err = row.Scan(&nextMonth)
	if err != nil {
		log.Println("Failed to db.Query:", err)
	}
	log.Println(nextMonth)

	return nextMonth
}

func getStandings(report string) string {
	var standings, sql string

	if report == "day" {
		standings = "```diff\nDaily Standings Report:\n"
		sql = "SELECT name, daily_win, daily_loss, (daily_win - daily_loss) as net_win FROM players WHERE daily_win != 0 OR daily_loss != 0 ORDER BY net_win DESC, daily_win DESC, name ASC;"
	} else if report == "week" {
		standings = "```diff\nWeekly Standings Report:\n"
		sql = "SELECT name, weekly_win, weekly_loss, (weekly_win - weekly_loss) as net_win FROM players WHERE weekly_win != 0 OR weekly_loss != 0 ORDER BY net_win DESC, weekly_win DESC, name ASC;"
	} else if report == "month" {
		standings = "```diff\nMonthly Standings Report:\n"
		sql = "SELECT name, monthly_win, monthly_loss, (monthly_win - monthly_loss) as net_win FROM players WHERE monthly_win != 0 OR monthly_loss != 0 ORDER BY net_win DESC, monthly_win DESC, name ASC;"
	}

	max := getStringArrayMaxLength(playerDb)

	for row, err := db.Query(sql); err == nil; err = row.Next() {
		var name string
		var win, loss int
		var net_win float64
		row.Scan(&name, &win, &loss, &net_win)

		sign := " "
		if win > loss {
			sign = "+"
		} else if win < loss {
			sign = "-"
		}

		standings += fmt.Sprintf("%s %s %s %d - %d  %s%d\n", sign, strings.Title(name), getPadLengthString(max, name), win, loss, sign, int(math.Abs(net_win)))
	}

	standings += "```"

	return standings
}

func getStreaks() string {
	streaks := "```diff\nCurrent Streaks:\n"
	sql := "SELECT name, streak FROM players WHERE streak > 1 OR streak < -1 ORDER BY streak DESC, name ASC"

	max := getStringArrayMaxLength(playerDb)

	for row, err := db.Query(sql); err == nil; err = row.Next() {
		var name string
		var streak float64
		row.Scan(&name, &streak)

		sign := " "
		if streak > 0 {
			sign = "+"
		} else {
			sign = "-"
		}

		streaks += fmt.Sprintf("%s %s %s %s%d\n", sign, strings.Title(name), getPadLengthString(max, name), sign, int(math.Abs(streak)))
	}

	streaks += "```"

	return streaks
}

func updateNextDay(db *sqlite3.Conn) {
	sql := "UPDATE settings SET value = datetime(value, '+24 hours') WHERE setting ='next_day';"
	db.Exec(sql)
}

func updateNextWeek(db *sqlite3.Conn) {
	sql := "UPDATE settings SET value = datetime(value, '+7 days') WHERE setting ='next_week';"
	db.Exec(sql)
}

func updateNextMonth(db *sqlite3.Conn) {
	sql := "UPDATE settings SET value = datetime(value, '+1 month') WHERE setting ='next_month';"
	db.Exec(sql)
}

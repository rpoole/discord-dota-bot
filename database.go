package main

import (
	"time"
	"log"
	"github.com/mxk/go-sqlite/sqlite3"
)

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

package main

import (
	"database/sql"
	"errors"
	"got/types"
	"strconv"
	"time"
)

type State struct {
	db *sql.DB

	LastCheckoutID uint64
	CurrentEntry   *types.Entry
	CurrentSheet   string
	LastSheet      string
}

func MakeState(db *sql.DB) (*State, error) {
	state := &State{
		db: db,
	}

	var err error

	state.CurrentEntry, err = state.GetCurrentEntry()
	if err != nil {
		return nil, err
	}

	meta, err := state.GetMeta()
	if err != nil {
		return nil, err
	}

	lastCheckoutID, err := strconv.ParseUint(meta["last_checkout_id"], 10, 64)
	if err != nil {
		return nil, err
	}

	state.CurrentSheet = meta["current_sheet"]
	state.LastSheet = meta["last_sheet"]
	state.LastCheckoutID = lastCheckoutID

	return state, nil
}

func (s *State) GetMeta() (map[string]string, error) {
	rows, err := s.db.Query("select key, value from meta")
	if err != nil {
		return nil, err
	}

	res := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		res[key] = value
	}

	return res, nil
}

func (s *State) GetEntry(id uint64) (*types.Entry, error) {
	row := s.db.QueryRow("select * from entries where id = ?", id)

	var e types.DatabaseEntry

	err := row.Scan(&e.ID, &e.Note, &e.Start, &e.End, &e.Sheet)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return e.ToEntry()
}

func (s *State) Close() error {
	return s.db.Close()
}

func (s *State) StartEntry(note, sheet string, start time.Time) (uint64, error) {
	current, err := s.GetCurrentEntry()
	if err != nil {
		return 0, err
	}
	if current != nil {
		return 0, errors.New("already running")
	}

	res, err := s.db.Exec("insert into entries(note, start, sheet) values(?, ?, ?)", note, start, sheet)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	return uint64(id), err
}
func (s *State) StopEntry(id uint64, end time.Time) error {
	entry, err := s.GetCurrentEntry()
	if err != nil {
		return err
	} else if entry == nil {
		return errors.New("not running")
	}

	if err := s.SetLastCheckoutId(id); err != nil {
		return err
	}

	_, err = s.db.Exec("update entries set end = ? where id = ?", end, id)
	return err
}

func (s *State) SetLastCheckoutId(id uint64) error {
	_, err := s.db.Exec("update meta set value = ? where key = ?", id, "last_checkout_id")
	if err != nil {
		return err
	}

	s.LastCheckoutID = id
	return nil
}

func (s *State) GetCurrentSheet() (string, error) {
	row := s.db.QueryRow("select value from meta where key = ?", "current_sheet")
	var res string
	err := row.Scan(&res)
	return res, err
}

func (s *State) GetCurrentEntry() (*types.Entry, error) {
	// HACK
	row := s.db.QueryRow("select id from entries where end is null")
	var id uint64
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	entry, err := s.GetEntry(id)
	return entry, err
}
func (s *State) GetLastEntry(sheet string) (*types.Entry, error) {
	entries, err := s.GetAllEntries(sheet)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	return entries[len(entries)-1], nil
}

func (s *State) GetAllEntries(sheetName string) ([]*types.Entry, error) {
	var res []*types.Entry

	var rows *sql.Rows
	var err error
	if sheetName != "" {
		rows, err = s.db.Query("select * from entries where sheet = ?", sheetName)
	} else {
		rows, err = s.db.Query("select * from entries")
	}
	if err != nil {
		return res, nil
	}

	for rows.Next() {
		var e types.DatabaseEntry
		err := rows.Scan(&e.ID, &e.Note, &e.Start, &e.End, &e.Sheet)
		if err != nil {
			return res, nil
		}

		entry, err := e.ToEntry()
		if err != nil {
			return res, err
		}
		res = append(res, entry)
	}

	return res, nil
}

func (s *State) SwitchSheet(sheet string) error {
	current := s.CurrentSheet

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	tx.Exec("update meta set value = ? where key = ?", current, "last_sheet")
	tx.Exec("update meta set value = ? where key = ?", sheet, "current_sheet")

	return tx.Commit()
}

func (s *State) GetAllSheets() ([]string, error) {
	var res []string

	rows, err := s.db.Query("select distinct sheet from entries")
	if err != nil {
		return res, err
	}

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return res, err
		}
		res = append(res, name)
	}

	return res, nil
}

func (s *State) RemoveSheet(name string) error {
	_, err := s.db.Exec("delete from entries where sheet = ?", name)
	return err
}

func (s *State) RemoveEntry(id uint64) error {
	_, err := s.db.Exec("delete from entries where id = ?", id)
	return err
}

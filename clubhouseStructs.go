package main

import "time"

type ClubHouseEpic struct {
	Id int64 `json:"id"`
}

type ClubHouseCreateEpic struct {
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Name        string    `json:"name"`
	key         string
}

type CreateComment struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type ClubHouseCreateStory struct {
	Comments    []CreateComment        `json:"comments"`
	CreatedAt   time.Time              `json:"created_at"`
	Description string                 `json:"description"`
	Estimate    int64                  `json:"estimate"`
	EpicId      int64                  `json:"epic_id"`
	Labels      []ClubHouseCreateLabel `json:"labels"`
	Name        string                 `json:"name"`
	ProjectId   int64                  `json:"project_id"`
	Tasks       []ClubHouseCreateTask  `json:"tasks"`
	StoryType   string                 `json:"story_type"`
	epicLink    string
	key         string
}

type ClubHouseCreateLabel struct {
	Name string `json:"name"`
}
type ClubHouseCreateTask struct {
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Complete    bool      `json:"complete"`
	parent      string
}

type ClubHouseData struct {
	Epics   []ClubHouseCreateEpic  `json:"epics"`
	Stories []ClubHouseCreateStory `json:"stories"`
}

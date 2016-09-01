package main

import "time"

// ClubHouseEpic is the data returned from the Clubhouse API when the Epic is created.
type ClubHouseEpic struct {
	ID int64 `json:"id"`
}

// ClubHouseCreateEpic is the object sent to the API to create an Epic. ClubHouseEpic is the return from the submission of this struct.
type ClubHouseCreateEpic struct {
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Name        string    `json:"name"`
	key         string
}

// ClubHouseCreateComment is used in ClubHouseCreateStory for comments.
type ClubHouseCreateComment struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// ClubHouseCreateStory is the object sent to API to submit a Story, Tasks, & Comment
type ClubHouseCreateStory struct {
	Comments    	[]ClubHouseCreateComment `json:"comments"`
	CreatedAt   	time.Time                `json:"created_at"`
	Description 	string                   `json:"description"`
	Estimate    	int64                    `json:"estimate"`
	EpicID      	int64                    `json:"epic_id,omitempty"`
	Labels      	[]ClubHouseCreateLabel   `json:"labels"`
	Name        	string                   `json:"name"`
	ProjectID   	int64                    `json:"project_id"`
	Tasks       	[]ClubHouseCreateTask    `json:"tasks"`
	StoryType   	string                   `json:"story_type"`
	epicLink    	string
	key         	string
	OwnerIDs		[]string				 `json:"owner_ids"`
	WorkflowState	int64					 `json:"workflow_state_id"`
}

// ClubHouseCreateLabel is used to submit labels with stories, it looks like from the API that duplicates will not be created.
type ClubHouseCreateLabel struct {
	Name string `json:"name"`
}

// ClubHouseCreateTask is used for Tasks in stories.
type ClubHouseCreateTask struct {
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Complete    bool      `json:"complete"`
	parent      string
}

// ClubHouseData is a container holding the data for submission of writing to a JSON file.
type ClubHouseData struct {
	Epics   []ClubHouseCreateEpic  `json:"epics"`
	Stories []ClubHouseCreateStory `json:"stories"`
}

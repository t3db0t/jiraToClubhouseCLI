package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/kennygrant/sanitize"
)

type JiraExport struct {
	ElementName xml.Name   `xml:"rss"`
	Items       []JiraItem `xml:"channel>item"`
}

type JiraItem struct {
	Assignee        string   `xml:"assignee"`
	CreatedAtString string   `xml:"created"`
	Description     string   `xml:"description"`
	Key             string   `xml:"key"`
	Labels          []string `xml:"labels>label"`
	Project         string   `xml:"project"`
	Resolution      string   `xml:"resolution"`
	Reporter        string   `xml:"reporter"`
	Status          string   `xml:"status"`
	Summary         string   `xml:"summary"`
	Title           string   `xml:"title"`
	Type            string   `xml:"type"`
	Parent          string   `xml:"parent"`

	Comments     []JiraComment      `xml:"comments>comment"`
	CustomFields []JiraCustomFields `xml:"customfields>customfield"`

	epicLink string
}

type JiraCustomFields struct {
	FieldName  string   `xml:"customfieldname"`
	FieldVales []string `xml:"customfieldvalues>customfieldvalue"`
}
type JiraComment struct {
	Author          string `xml:"author,attr"`
	CreatedAtString string `xml:"created,attr"`
	Comment         string `xml:",chardata"`
	Id              string `xml:"id,attr"`
}

func (je *JiraExport) GetDataForClubhouse(projectId int64) ClubHouseData {
	epics := []JiraItem{}
	tasks := []JiraItem{}
	stories := []JiraItem{}
	for _, item := range je.Items {
		switch item.Type {
		case "Epic":
			epics = append(epics, item)
			break
		case "Sub-task":
			tasks = append(tasks, item)
			break
		default:
			stories = append(stories, item)
			break
		}
	}
	chEpics := []ClubHouseCreateEpic{}
	for _, item := range epics {
		chEpics = append(chEpics, item.CreateEpic())
	}
	chTasks := []ClubHouseCreateTask{}
	chStories := []ClubHouseCreateStory{}
	for _, item := range tasks {
		chTasks = append(chTasks, item.CreateTask())
		// if item.Description != "" || len(item.Comments) > 0 {
		// 	chStories = append(chStories, item.CreateStory(projectId))
		// } else {
		// 	chTasks = append(chTasks, item.CreateTask())
		// }
	}
	for _, item := range stories {
		chStories = append(chStories, item.CreateStory(projectId))
	}
	storyMap := make(map[string]int)
	for i, item := range chStories {
		storyMap[item.key] = i
	}
	for _, task := range chTasks {
		chStories[storyMap[task.parent]].Tasks = append(chStories[storyMap[task.parent]].Tasks, task)
	}
	return ClubHouseData{Epics: chEpics, Stories: chStories}
}

func (item *JiraItem) CreateEpic() ClubHouseCreateEpic {
	return ClubHouseCreateEpic{Description: sanitize.HTML(item.Description), Name: sanitize.HTML(item.Summary), key: item.Key, CreatedAt: ParseJiraTimeStamp(item.CreatedAtString)}
}

func (item *JiraItem) CreateTask() ClubHouseCreateTask {
	return ClubHouseCreateTask{Description: sanitize.HTML(item.Summary), parent: item.Parent, Complete: false}
}

func (item *JiraItem) CreateStory(projectId int64) ClubHouseCreateStory {
	comments := []CreateComment{}
	for _, c := range item.Comments {
		comments = append(comments, c.CreateComment())
	}
	labels := []ClubHouseCreateLabel{}
	for _, label := range item.Labels {
		labels = append(labels, ClubHouseCreateLabel{Name: strings.ToLower(label)})
	}

	return ClubHouseCreateStory{
		Comments:    comments,
		CreatedAt:   ParseJiraTimeStamp(item.CreatedAtString),
		Description: sanitize.HTML(item.Description),
		Labels:      labels,
		Name:        sanitize.HTML(item.Summary),
		ProjectId:   projectId,
		StoryType:   item.GetClubhouseType(),
		key:         item.Key,
		epicLink:    item.GetEpicLink()}
}

func (comment *JiraComment) CreateComment() CreateComment {
	return CreateComment{Text: fmt.Sprintf("%s: %s", comment.Author, sanitize.HTML(comment.Comment)), CreatedAt: ParseJiraTimeStamp(comment.CreatedAtString)}
}

func (item *JiraItem) GetEpicLink() string {
	for _, cf := range item.CustomFields {
		if cf.FieldName == "Epic Link" {
			return cf.FieldVales[0]
		}
	}
	return ""
}

func (item *JiraItem) GetClubhouseType() string {
	if item.Type == "Bug" {
		return "bug"
	}
	return "feature"
}

func ParseJiraTimeStamp(dateString string) time.Time {
	format := "Mon, 2 Jan 2006 15:04:05 -0700"
	t, err := time.Parse(format, dateString)
	if err != nil {
		return time.Now()
	}
	return t
}

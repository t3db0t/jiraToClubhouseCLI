package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/kennygrant/sanitize"
)

// JiraExport is the container of Jira Items from the XML.
type JiraExport struct {
	ElementName xml.Name   `xml:"rss"`
	Items       []JiraItem `xml:"channel>item"`
}

// JiraItem is the struct for a basic item imported from the XML
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

	Comments     []JiraComment     `xml:"comments>comment"`
	CustomFields []JiraCustomField `xml:"customfields>customfield"`

	epicLink string
}

//JiraCustomField is the information for custom fields. Right now the only one used is the Epic Link
type JiraCustomField struct {
	FieldName  string   `xml:"customfieldname"`
	FieldVales []string `xml:"customfieldvalues>customfieldvalue"`
}

// JiraComment is a comment from the imported XML
type JiraComment struct {
	Author          string `xml:"author,attr"`
	CreatedAtString string `xml:"created,attr"`
	Comment         string `xml:",chardata"`
	ID              string `xml:"id,attr"`
}

//GetDataForClubhouse will take the data from the XML and translate it into a format for sending to Clubhouse
func (je *JiraExport) GetDataForClubhouse(projectID int64) ClubHouseData {
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
	}

	for _, item := range stories {
		chStories = append(chStories, item.CreateStory(projectID))
	}

	// storyMap is used to link the JiraItem's key to its index in the chStories slice. This is then used to assign subtasks properly
	storyMap := make(map[string]int)
	for i, item := range chStories {
		storyMap[item.key] = i
	}

	for _, task := range chTasks {
		chStories[storyMap[task.parent]].Tasks = append(chStories[storyMap[task.parent]].Tasks, task)
	}

	return ClubHouseData{Epics: chEpics, Stories: chStories}
}

// CreateEpic returns a ClubHouseCreateEpic from the JiraItem
func (item *JiraItem) CreateEpic() ClubHouseCreateEpic {
	return ClubHouseCreateEpic{Description: sanitize.HTML(item.Description), Name: sanitize.HTML(item.Summary), key: item.Key, CreatedAt: ParseJiraTimeStamp(item.CreatedAtString)}
}

// CreateTask returns a task if the item is a Jira Sub-task
func (item *JiraItem) CreateTask() ClubHouseCreateTask {
	return ClubHouseCreateTask{Description: sanitize.HTML(item.Summary), parent: item.Parent, Complete: false}
}

// CreateStory returns a ClubHouseCreateStory from the JiraItem
func (item *JiraItem) CreateStory(projectID int64) ClubHouseCreateStory {
	comments := []ClubHouseCreateComment{}
	for _, c := range item.Comments {
		comments = append(comments, c.CreateComment())
	}

	labels := []ClubHouseCreateLabel{}
	for _, label := range item.Labels {
		labels = append(labels, ClubHouseCreateLabel{Name: strings.ToLower(label)})
	}

	/* Overwrite supplied Project ID
	   if assigned to loz, Frontend-Import
	   if item.Summary starts with "FE: ", Frontend-Import
	   if item.Summary contains "Touch", Touch
	   etc

	- "Backend-Import": 241
	- "Frontend-Import": 242
	- "Gateway-Import": 243
	- Touch: 19

	*/

	// Map JIRA assignee to Clubhouse owner(s)
	// var owners []string
	fmt.Println(item.Assignee)
	// switch item.Assignee {
	//     case "10200":
	//         // ready for test
	//         state := 500000010
	//     case "3":
	//         // in progress
	//         state := 500000015
	//     case "10101":
	//     	// selected
	//     	state := 500000011
	//     case "10100":
	//     	// backlog
	//         state := 500000014
	//     case "":
	//     	// done/completed
	//     	state := 500000012
	//     default:
	//     	// backlog
	//         state := 500000014
 //    }
	// append(owners, owner)

	// Map JIRA status to Clubhouse Workflow state
	// cases break automatically, no fallthrough by default
	fmt.Println(item.Status)
	var state int64 = 500000014
	switch item.Status {
	    case "10200":
	        // ready for test
	        state = 500000010
	    case "3":
	        // in progress
	        state = 500000015
	    case "10101":
	    	// selected
	    	state = 500000011
	    case "10100":
	    	// backlog
	        state = 500000014
	    case "":
	    	// done/completed
	    	state = 500000012
	    default:
	    	// backlog
	        state = 500000014
    }

	return ClubHouseCreateStory{
		Comments:    	comments,
		CreatedAt:   	ParseJiraTimeStamp(item.CreatedAtString),
		Description: 	sanitize.HTML(item.Description),
		Labels:      	labels,
		Name:        	sanitize.HTML(item.Summary),
		ProjectID:   	projectID,
		StoryType:   	item.GetClubhouseType(),
		key:         	item.Key,
		epicLink:    	item.GetEpicLink(),
		WorkflowState:	state,
		// OwnerIDs:		owners
	}
}

// CreateComment takes the JiraItem's comment data and returns a ClubHouseCreateComment
func (comment *JiraComment) CreateComment() ClubHouseCreateComment {
	return ClubHouseCreateComment{Text: fmt.Sprintf("%s: %s", comment.Author, sanitize.HTML(comment.Comment)), CreatedAt: ParseJiraTimeStamp(comment.CreatedAtString)}
}

// GetEpicLink returns the Epic Link of a Jira Item.
func (item *JiraItem) GetEpicLink() string {
	for _, cf := range item.CustomFields {
		if cf.FieldName == "Epic Link" {
			return cf.FieldVales[0]
		}
	}
	return ""
}

// GetClubhouseType determines type based on if the Jira item is a bug or not.
func (item *JiraItem) GetClubhouseType() string {
	if item.Type == "Bug" {
		return "bug"
	}
	return "feature"
}

// ParseJiraTimeStamp parses the format in the XML using Go's magical timestamp.
func ParseJiraTimeStamp(dateString string) time.Time {
	format := "Mon, 2 Jan 2006 15:04:05 -0700"
	t, err := time.Parse(format, dateString)
	if err != nil {
		return time.Now()
	}
	return t
}

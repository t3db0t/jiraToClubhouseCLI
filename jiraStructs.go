package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
	"strconv"

	"github.com/kennygrant/sanitize"
)

// JiraExport is the container of Jira Items from the XML.
type JiraExport struct {
	ElementName xml.Name   `xml:"rss"`
	Items       []JiraItem `xml:"channel>item"`
}

type JiraAssignee struct {
	Username		string	`xml:"username,attr"`
}

type JiraReporter struct {
	Username		string	`xml:"username,attr"`
}

// JiraItem is the struct for a basic item imported from the XML
type JiraItem struct {
	Assignee        JiraAssignee   `xml:"assignee"`
	CreatedAtString string   		`xml:"created"`
	Description     string   		`xml:"description"`
	Key             string   		`xml:"key"`
	Labels          []string 		`xml:"labels>label"`
	Project         string   		`xml:"project"`
	Resolution      string   		`xml:"resolution"`
	Reporter        JiraReporter  	`xml:"reporter"`
	Status          string   		`xml:"status"`
	Summary         string   		`xml:"summary"`
	Title           string   		`xml:"title"`
	Type            string   		`xml:"type"`
	Parent          string   		`xml:"parent"`

	Comments     []JiraComment     `xml:"comments>comment"`
	CustomFields []JiraCustomField `xml:"customfields>customfield"`

	epicLink string
	UpdatedAtString string   		`xml:"updated"`
	ResolvedAtString string   		`xml:"resolved"`
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

func GetUserInfo(userMaps []userMap, jiraUsername string) (CHProjectID int, CHID string) {
	for _, u := range userMaps {
		if u.JiraUsername == jiraUsername {
			return u.CHProjectID, u.CHID
		}
	}
	return 0, ""
}

//GetDataForClubhouse will take the data from the XML and translate it into a format for sending to Clubhouse
func (je *JiraExport) GetDataForClubhouse(userMaps []userMap) ClubHouseData {
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
		chStories = append(chStories, item.CreateStory(userMaps))
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
func (item *JiraItem) CreateStory(userMaps []userMap) ClubHouseCreateStory {
	// fmt.Println("assignee: ", item.Assignee, "reporter: ", item.Reporter)
	// return ClubHouseCreateStory{}

	comments := []ClubHouseCreateComment{}
	for _, c := range item.Comments {
		comments = append(comments, c.CreateComment(userMaps))
	}

	labels := []ClubHouseCreateLabel{}
	for _, label := range item.Labels {
		labels = append(labels, ClubHouseCreateLabel{Name: strings.ToLower(label)})
	}

	// Add a label for tracking jira sprints in 
	lastSprint := item.GetLastSprint()

	if lastSprint != "" {
		labels = append(labels, ClubHouseCreateLabel{Name: strings.ToLower(lastSprint)})
	}

	// Adding special label that indicates that it was imported from JIRA
	labels = append(labels, ClubHouseCreateLabel{Name: "jira"})

	// Overwrite supplied Project ID
	projectID := MapProject(userMaps, item.Assignee.Username)
	// projectID, ownerID := GetUserInfo(userMaps, item.Assignee.Username)

	// Map JIRA assignee to Clubhouse owner(s)
	// Leave array empty if username is unknown
	// Must use "make" function to force empty array for correct JSON marshalling
	ownerID := MapUser(userMaps, item.Assignee.Username)
	var owners []string
	if ownerID != "" {
		// owners := []string{ownerID}
		owners = append(owners, ownerID)
	} else {
		owners = make([]string, 0)
	}

	// Map JIRA status to Clubhouse Workflow state
	// cases break automatically, no fallthrough by default
	var state int64 = 500000014
	switch item.Status {
	    case "Ready for Test":
	        // ready for test
	        state = 500000010
	    case "Task In Progress":
	        // in progress
	        state = 500000015
	    case "Selected for Review/Development":
	    	// selected
	    	state = 500000011
	    case "Task backlog":
	    	// backlog
	        state = 500000014
	    case "Done":
	    	// Completed
	    	state = 500000012
	    case "Verified":
	    	// Completed
	    	state = 500000012
	    case "Closed":
	    	state = 500000021
	    default:
	    	// backlog
	        state = 500000014
    }

    requestor := MapUser(userMaps, item.Reporter.Username)
    // _, requestor := GetUserInfo(userMaps, item.Reporter.Username)
    if requestor == "" {
    	// map to me if requestor not in Clubhouse
    	requestor = MapUser(userMaps, "ted")
    	// _, requestor = GetUserInfo(userMaps, "ted")
    }

    fmt.Printf("%s: JIRA Assignee: %s | Project: %d | Status: %s\n\n", item.Key, item.Assignee.Username, projectID, item.Status)

	return ClubHouseCreateStory{
		Comments:    	comments,
		CreatedAt:   	ParseJiraTimeStamp(item.CreatedAtString),
		UpdatedAt:   	ParseJiraTimeStamp(item.UpdatedAtString),
		CompletedAt:   	ParseJiraTimeStamp(item.ResolvedAtString),
		StartedAt:   	ParseJiraTimeStampWithDelta(item.ResolvedAtString, -1),
		Description: 	sanitize.HTML(item.Description),
		Labels:      	labels,
		Name:        	sanitize.HTML(item.Summary),
		ProjectID:   	int64(projectID),
		StoryType:   	item.GetClubhouseType(),
		key:         	item.Key,
		epicLink:    	item.GetEpicLink(),
		WorkflowState:	state,
		OwnerIDs:		owners,
		RequestedBy:	requestor,
		Estimate: 		item.GetEstimate(),
	}
}

func MapUser(userMaps []userMap, jiraUserName string) string {
	_, chUserID := GetUserInfo(userMaps, jiraUserName)

	if chUserID == "" {
		fmt.Println("[MapUser] JIRA user not found: ", jiraUserName)
    	return ""
	}

	return chUserID
}

func MapProject(userMaps []userMap, jiraUserName string) int {
	projectID, _ := GetUserInfo(userMaps, jiraUserName)

	if projectID == 0 {
		fmt.Println("[MapProject] JIRA user not found: ", jiraUserName)
    	return 299
	}

	return projectID
}


// CreateComment takes the JiraItem's comment data and returns a ClubHouseCreateComment
func (comment *JiraComment) CreateComment(userMaps []userMap) ClubHouseCreateComment {
	commentText := sanitize.HTML(comment.Comment)
	if commentText == "\n" {
		commentText = "(empty)"
	}
	author := MapUser(userMaps, comment.Author)
	if author == "" {
		// since we MUST have a comment author, make it me and prepend the actual username to the comment body
		author = MapUser(userMaps, "ted")
		commentText = comment.Author + ": " + commentText
	}

	return ClubHouseCreateComment{
		Text:		commentText,
		CreatedAt:	ParseJiraTimeStamp(comment.CreatedAtString),
		Author: 	author,
	}
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

// GetEstimate returns the estimate of a Jira Item.
func (item *JiraItem) GetEstimate() int64 {
	for _, cf := range item.CustomFields {
		fmt.Println(cf.FieldName)
		if cf.FieldName == "Story point estimate" {
			fmt.Println(cf.FieldVales[0])
			if i, err := strconv.ParseFloat(cf.FieldVales[0], 64); err == nil {
				fmt.Println(int64(i))
				return int64(i)
			}
			
		}
	}
	return 0
}

// GetLastSprint returns the latest sprint a Jira Item was in.
func (item *JiraItem) GetLastSprint() string {
	for _, cf := range item.CustomFields {
		fmt.Println(cf.FieldName)
		if cf.FieldName == "Sprint" && len(cf.FieldVales) > 0 {
			return cf.FieldVales[len(cf.FieldVales)-1]
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
func ParseJiraTimeStampWithDelta(dateString string, daysToAdd int) time.Time {
	format := "Mon, 2 Jan 2006 15:04:05 -0700"
	t, err := time.Parse(format, dateString)
	if err != nil {
		return time.Now().AddDate(0, 0, daysToAdd)
	}
	return t.AddDate(0, 0, daysToAdd)
}

// ParseJiraTimeStamp parses the format in the XML using Go's magical timestamp.
func ParseJiraTimeStamp(dateString string) time.Time {
	return ParseJiraTimeStampWithDelta(dateString, 0)
}

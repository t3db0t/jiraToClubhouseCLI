# jiraToClubhouseCLI

[Clubhouse.io](http://clubhouse.io) is a cool project management tool. The purpose of this tool is to help export your data out of Jira if you making the switch.

Updated by T3db0t to use an external JSON file to map user IDs and project IDs.  See this blog post for more information: [http://log.liminastudio.com/programming/how-and-why-i-switched-from-jira-to-clubhouse]

The tool has two modes: `export` and `import`.  Use `export` to generate a JSON file and `import` to actually send all the data to Clubhouse using your API token.  Generate an API token from `https://app.clubhouse.io/[your_organization]/settings/account/api-tokens` and make it an environment variable like so: `export CLUBHOUSE_API_TOKEN=asdflkjaseflkjdf`

Also included are a few utility scripts in Python: `createTestStory.py`, `deleteArchivedStories.py` and `deleteEmptyEpics.py`.  Use these like so:

`python deleteArchivedStories.py $CLUBHOUSE_API_TOKEN`

Note: The importer will add a "JIRA" label to every ticket it imports.  If something gets messed up and you need to redo an import, this makes it easy to select all imported tickets, archive them and use the above command to delete them.

## User Map

A JSON file with this structure:

```
[
	{
		"jiraUsername": "userA",
		"chProjectID": 5,
		"chID": "476257c9-ac5a-46bc-67d6-4bc8bbfde7be"
	},
	{
		"jiraUsername": "userB",
		"chProjectID": 81,
		"chID": "476257c9-ac5a-46bc-67d6-4bc8bbfde7be"
	},
	{
		"jiraUsername": "userC",
		"chProjectID": 6,
		"chID": "476257c9-ac5a-46bc-67d6-4bc8bbfde7be"
	}
]
```

You'll need to get the right JIRA usernames and Clubhouse Project ID and user IDs and fill them in with this structure.  Detail on how to do this in [my blog post](http://log.liminastudio.com/programming/how-and-why-i-switched-from-jira-to-clubhouse).

## Export
To see what the data will look like (as JSON).

```bash
go run *.go export --in SearchRequest.xml --map userMap.json --out file.json
```


### Params
 * `--in` The xml file you want to read from
 * `--map` The user maps
 * `--out` The file you want to export the JSOn to.

## Import

To actually import to Clubhouse. Use `--test` to parse input files and 'preview' what the effects will be without uploading to Clubhouse.  Good for checking that everything works before pulling the trigger.

```bash
go run *.go import --in SearchRequest.xml --map userMap.json --token $CLUBHOUSE_API_TOKEN
```

### Params
 * `--in` The xml file you want to read from
 * `--map` The user maps
 * `--token` The api token for your Clubhouse instance
 * `--test` Test mode: run the program, but do not upload to Clubhouse.
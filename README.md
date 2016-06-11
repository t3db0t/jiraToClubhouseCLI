# jiraToClubhouseCLI

[Clubhouse.io](http://clubhouse.io) is project management tool. The purpose of this tool is to help export your data out of Jira if you making the switch.

## Export
To see what the data will sort of look like.

```bash
go run *.go export --in SearchRequest.xml --projectId 5 --out file.json
```


### Params
 * `--in` The xml file you want to read from
 * `--projectId` the project Id you want to use for the items you are importing
 * `--out` The file you want to export the JSOn to.

## Import

To actually import to Clubhouse

```bash
go run *.go import --in SearchRequest.xml --projectId 5 --token jgjjgjgkkjgjkgjk
```

### Params
 * `--in` The xml file you want to read from
 * `--projectId` the project Id you want to use for the items you are importing
 * `--token` The api token for your Clubhouse instance

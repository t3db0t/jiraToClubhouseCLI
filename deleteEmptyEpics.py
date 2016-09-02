# Clubhouse.io: Delete all epics with no stories

import requests
import sys
import json

token = sys.argv[1]

try:
	r = requests.get('https://api.clubhouse.io/api/v1/epics', params={'token':token})
except RequestException as err:
	print err

epics = r.json()

epic_ids = [e['id'] for e in epics]
print "Found %d epics" % len(epic_ids)

for eid in epic_ids:
	try:
		search = {'epic_id':eid}
		r = requests.post('https://api.clubhouse.io/api/v1/stories/search', params={'token':token}, data=search)
	except RequestException as err:
		print err

	stories = r.json()
	print "Found %d stories in epic #%d" % (len(stories), eid)

	if(len(stories) == 0):
		# delete this epic
		url = 'https://api.clubhouse.io/api/v1/epics/%d' % eid
		print "Deleting epic #%d" % eid
		r = requests.delete(url, params={'token':token})
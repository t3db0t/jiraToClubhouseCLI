# Clubhouse.io: Delete all archived stories

import requests
import sys
import json

token = sys.argv[1]

try:
	search = {'archived':'true'}
	r = requests.post('https://api.clubhouse.io/api/v2/stories/search', params={'token':token}, data=search)
except RequestException as err:
	print err

archivedStories = r.json()

print archivedStories

print "Found %d archived stories" % len(archivedStories)

for story in archivedStories:
	print 'Deleting Story #%d' % story['id']
	url = 'https://api.clubhouse.io/api/v2/stories/%d' % story['id']
	try:
		r = requests.delete(url, params={'token':token})
	except RequestException as err:
		print err
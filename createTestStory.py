# Clubhouse.io: Delete all archived stories

import requests
import sys
import json

token = sys.argv[1]

try:
	data = {
		'name':'TEST STORY',
		'project_id':299,
		'owner_ids':[]
	}
	r = requests.post('https://api.clubhouse.io/api/v1/stories', params={'token':token}, data=data)
	print r.text()
except RequestException as err:
	print err
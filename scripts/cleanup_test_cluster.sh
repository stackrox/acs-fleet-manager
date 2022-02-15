#!/bin/bash

# This bash script is used to clean up an OSD cluster created and used by integration tests
# against real OCM environment. It uses a internal/dinosaur/test/integration/test_cluster.json file with cluster details created by the tests.

FILE=internal/dinosaur/test/integration/test_cluster.json
INTEGRATION_ENV="integration"
ENV="$OCM_ENV"
if test -f "$FILE"; then
  if [[ $ENV != "$INTEGRATION_ENV" ]] ; then
    # Dump file to be able to inspect content during debugging
    echo "Dumping content of $FILE:"
    cat $FILE
    # Retrieve cluster id
    clusterID=$(jq -r .cluster_id ${FILE})
    if [[ $clusterID != "null" ]] ; then
      ocm login --url=https://api.stage.openshift.com/ --token="$OCM_OFFLINE_TOKEN"
      ocm delete /api/clusters_mgmt/v1/clusters/"$clusterID"
      rm $FILE
    else 
      echo "Cluster id not found in file $FILE"
      exit 1  
    fi
  fi
else 
  echo "File $FILE does not exist"
  exit 1  
fi

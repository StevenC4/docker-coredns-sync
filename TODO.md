# Todo

## Features
* Better dev environment
  * Decouple from production etcd instance (maybe a docker-compose file for the dev environment that defines the devcontainer as well as dependencies - an etcd instance, a coredns instance, etc.)
    * May have some ways to preconfigure different datasets into the etcd instance (either automatically or via makefile scripts - as well as an easy way to reset the etcd state)
	* This way, I can test it and do development on my macbook. That way I don't have to stop the prod instance, spin up a dev container on the remote prod instance, and test it against the production etcd instance. This avoids impact on my production DNS resolution

## Edge cases
* Scenario 1:
  1. Spin up dev docker-coredns-sync configured with HOST_IP=192.168.1.5 (incorrect value in this hypothetical scenario)
  2. Spin up another container (container A) that has an A record but doesn't have a label configured for the IP address - this results in the sync using the HOST_IP env var value as the IP address of the A record
  3. Change the HOST_IP to 192.168.1.6 (correct value)
  4. Spin down and back up the docker-coredns-sync container for the new HOST_IP address to be used
  5. On startup, we pull all keys from etcd, pull a list of all containers on our host, then see which records already exist. We get to container A, which defines an A record. We detect that we already have one and don't take action. However, the existing records has the wrong IP address now that we've changed the IP address env var, but our app doesn't detect that and fix it. Or if we spin down container A and spin it back up, we try to deregister its A record but can't, because the IP address of the record from etcd doesn't match the IP address in the record inferred from the docker labels. Then when we spin container A back up, it once again detects a collision or that it's owned by our host - either way, it doesn't try to update the record.
  * I want to implement a way to correct these records automatically without going through a cumbersome process.

## Bugs
* 

## Updates
* Find and fix code smells

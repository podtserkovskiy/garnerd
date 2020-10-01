# GarnerD
A dynamic cache for docker images in minikube.

## ToDo:
- Timeouts
- Eviction by size
- Update cache in the storage if ImageID has changed
- Support container creation events (kubernetes creates containers using image-id, but pulls using tags)
- Statistics
- Research LFU or ARC eviction

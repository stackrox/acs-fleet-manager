# rhacs-terraform add-on

## Quickstart

All following commands should be run from this directory.

### Build and run operator as a local process

```bash
make install run
```

### Build and run operator as a k8s deployment

```bash
docker login quay.io

# build and publish operator image
make docker-build docker-push

# launch deployment to cluster
make install deploy
# check deployment is running fine
kubectl -n rhacs-terraform-system get deployment rhacs-terraform-controller-manager -w
# tail controller logs
make logs/controller

# cleanup
make undeploy
```

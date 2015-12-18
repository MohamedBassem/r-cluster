# R-Cluster

A simple go server to schedule R scripts on a mesos cluster.

### What is this?

Refer to this medium post and this [deployment template repo](https://github.com/MohamedBassem/azure-rconsole-template) for details and use cases.

### Deployment

The tool needs a special architecture to run on :
- A mesos cluster.
- Docker installed on all the machines.
- A shared storage (e.g. NFS) between all the cluster nodes mounted at `/mnt/nfs`.

Then on the master node run :

```bash
go get https://github.com/MohamedBassem/r-cluster
cd $GOPATH/src/github.com/MohamedBassem/r-cluster
./r-cluster
```

### TODO

- A better UI.
- Making the tool configurable using a config file instead of hardcoding configurations.
- Basic Auth.
- Open Issues.

### Contribution

Your contributions and ideas are welcomed through issues and pull requests.

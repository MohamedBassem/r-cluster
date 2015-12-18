# R-Cluster

A simple go server to schedule R scripts on a mesos cluster.

### What is this?

Refer to this medium post and this [deployment template](https://github.com/MohamedBassem/azure-rconsole-template) for the details and use case.

### Deployment

The tool needs a special architecture to run on :

- A mesos cluster.
- Docker installed on all the machines.
- A shared storage (e.g. NFS) between all the cluster nodes mounted at `/mnt/nfs`.

### TODO

- A better UI.
- Making the tool configurable using a config file instead of hardcoding configurations.
- Open Issues.

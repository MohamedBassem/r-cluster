#!/bin/bash

WORKDIR_ROOT=/mnt/nfs/working_dir


print_usage() {
  echo "./run-r-script.sh --name name --command command"
}

while [[ $# > 1 ]]
do
key="$1"

case $key in
    --name)
    NAME="$2"
    shift
    ;;
    --command)
    COMMAND="$2"
    shift
    ;;
    *)
    echo "INVALID ARGUMENT $key"
    print_usage
    exit 1
    ;;
esac
shift # past argument or value
done

if [ -z "$NAME" ] || [ -z "$COMMAND" ]  ; then
  echo "Missing arguments"
  print_usage
  exit 1
fi

COMMAND="docker run -v $WORKDIR_ROOT/$NAME/input:/input -v $WORKDIR_ROOT/$NAME/output:/output -v $WORKDIR_ROOT/$NAME/code:/code 192.168.1.204:5011/bundled-r $COMMAND"

MASTER=`mesos-resolve $(cat /etc/mesos/zk)`
TASK_NAME=`cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1`

mesos-execute --master=$MASTER --name=$TASK_NAME --command="$COMMAND" --resources="cpus:4;mem:2048" > /dev/null 2>&1 &
EXECUTE_PID=`echo $!`


LINES_READ=0
while kill -0 $EXECUTE_PID 2> /dev/null; do
  NEW_LINES=`mesos-cat -i $TASK_NAME stdout stderr | tail -n +$LINES_READ`
  NEW_LINES_C=`echo -e "$NEW_LINES" | wc -l`
  LINES_READ=$((LINES_READ + NEW_LINES_C))
  echo -en "$NEW_LINES"
  sleep 0.5
done

sleep 5
mesos-cat -i $TASK_NAME stdout stderr | tail -n +$LINES_READ

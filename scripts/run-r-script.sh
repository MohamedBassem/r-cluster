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


TASK_NAME=`cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1`

mkdir -p /$WORKDIR_ROOT/$NAME/tmp
echo "#!/bin/bash" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "echo -e '\n=======================> Output Start <======================='" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "$COMMAND" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "echo -e '=======================> Output End <=======================\n'" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh

COMMAND="docker run --rm -v $WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh:/tmp/cmd.sh -v $WORKDIR_ROOT/$NAME/input:/input -v $WORKDIR_ROOT/$NAME/output:/output -v $WORKDIR_ROOT/$NAME/code:/code 192.168.1.204:5011/bundled-r /bin/bash /tmp/cmd.sh"

MASTER=`mesos-resolve $(cat /etc/mesos/zk)`

mesos-execute --master=$MASTER --name=$TASK_NAME --command="$COMMAND" --resources="cpus:4;mem:2048" > /dev/null 2>&1 &
EXECUTE_PID=`echo $!`


LINES_READ=0
PREV=""
set -x
while kill -0 $EXECUTE_PID 2> /dev/null; do
  PREV_C=$(echo -en "$PREV" | wc -c)
  TOTAL=`mesos-cat -i $TASK_NAME stdout`
  TOTAL_C=$(echo -en "$TOTAL" | wc -c)
  echo -en "$TOTAL" | tail -c $((TOTAL_C - PREV_C))
  PREV="$TOTAL"
  sleep 0.5
done
set +x

sleep 5
PREV_C=$(echo -en "$PREV" | wc -c)
TOTAL=`mesos-cat -i $TASK_NAME stdout`
TOTAL_C=$(echo -en "$TOTAL" | wc -c)
echo -en "$TOTAL" | tail -c $((TOTAL_C - PREV_C))

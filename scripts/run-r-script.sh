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


export TASK_NAME=`cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1`

mkdir -p /$WORKDIR_ROOT/$NAME/tmp
echo "#!/bin/bash" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "echo -e '\n=======================> Output Start <======================='" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "$COMMAND" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "echo -e '=======================> Output End <=======================\n'" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh

COMMAND="docker run --rm -w /task-dir -v $WORKDIR_ROOT/$NAME:/task-dir -v $WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh:/tmp/cmd.sh -v $WORKDIR_ROOT/$NAME/input:/input -v $WORKDIR_ROOT/$NAME/output:/output -v $WORKDIR_ROOT/$NAME/code:/code saherneklawy/r-datascience-docker /bin/bash /tmp/cmd.sh"

MASTER=`mesos-resolve $(cat /etc/mesos/zk)`

mkdir -p tmp
mkfifo tmp/$TASK_NAME
function execute_job {
  mesos-execute --master=$MASTER --name=$TASK_NAME --command="$COMMAND" --resources="cpus:1;mem:2048" | grep "Framework registered with" | awk '{print $4}' > tmp/$TASK_NAME
}

execute_job > /dev/null 2>&1 &
EXECUTE_PID=`echo $!`

FRAMEWORK_NAME=`cat tmp/$TASK_NAME`

LINES_READ=0
PREV=""
while kill -0 $EXECUTE_PID 2> /dev/null; do
  PREV_C=$(echo -en "$PREV" | wc -c)
  TOTAL=`mesos-cat --master=$MASTER --framework=$FRAMEWORK_NAME --task=$TASK_NAME --file=stdout`
  TOTAL_C=$(echo -en "$TOTAL" | wc -c)
  echo -en "$TOTAL" | tail -c $((TOTAL_C - PREV_C))
  PREV="$TOTAL"
  sleep 0.5
done

sleep 5
PREV_C=$(echo -en "$PREV" | wc -c)
TOTAL=`mesos-cat --master=$MASTER --framework=$FRAMEWORK_NAME --task=$TASK_NAME --file=stdout`
TOTAL_C=$(echo -en "$TOTAL" | wc -c)
echo -en "$TOTAL" | tail -c $((TOTAL_C - PREV_C))
rm tmp/$TASK_NAME

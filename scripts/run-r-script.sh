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
echo "echo -e '\n=======================> Output Start <=======================' >&2" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "$COMMAND" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "echo -e '=======================> Output End <=======================\n'" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh
echo "echo -e '=======================> Output End <=======================\n' >&2" >> /$WORKDIR_ROOT/$NAME/tmp/${TASK_NAME}.sh

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

OUT_COUNT=0
ERR_COUNT=0
while kill -0 $EXECUTE_PID 2> /dev/null; do

  ERR_TOTAL=`mesos-cat --master=$MASTER --framework=$FRAMEWORK_NAME --task=$TASK_NAME --file=stderr`
  echo -en "$ERR_TOTAL" | tail -c +$((ERR_COUNT + 1)) >&2
  echo -en "$ERR_TOTAL" | tail -c +$((ERR_COUNT + 1)) >> /$WORKDIR_ROOT/$NAME/stdfiles/${TASK_NAME}_stderr.txt
  ERR_COUNT=$(echo -en "$ERR_TOTAL" | wc -c)


  OUT_TOTAL=`mesos-cat --master=$MASTER --framework=$FRAMEWORK_NAME --task=$TASK_NAME --file=stdout`
  echo -en "$OUT_TOTAL" | tail -c +$((OUT_COUNT + 1))
  echo -en "$OUT_TOTAL" | tail -c +$((OUT_COUNT + 1)) >> /$WORKDIR_ROOT/$NAME/stdfiles/${TASK_NAME}_stdout.txt
  OUT_COUNT=$(echo -en "$OUT_TOTAL" | wc -c)

  sleep 0.1
done

sleep 5


ERR_TOTAL=`mesos-cat --master=$MASTER --framework=$FRAMEWORK_NAME --task=$TASK_NAME --file=stderr`
echo -en "$ERR_TOTAL" | tail -c +$((ERR_COUNT + 1)) >&2
echo -en "$ERR_TOTAL" | tail -c +$((ERR_COUNT + 1)) >> /$WORKDIR_ROOT/$NAME/stdfiles/${TASK_NAME}_stderr.txt

OUT_TOTAL=`mesos-cat --master=$MASTER --framework=$FRAMEWORK_NAME --task=$TASK_NAME --file=stdout`
echo -en "$OUT_TOTAL" | tail -c +$((OUT_COUNT + 1))
echo -en "$OUT_TOTAL" | tail -c +$((OUT_COUNT + 1)) >> /$WORKDIR_ROOT/$NAME/stdfiles/${TASK_NAME}_stdout.txt

rm tmp/$TASK_NAME

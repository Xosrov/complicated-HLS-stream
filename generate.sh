#!/bin/bash

USAGE="Usage: generate.sh INPUT IPADDR\nExample: generate.sh video.mp4 http://localhost:8080"

INPUTFILE=$1
IPADDR=$2

if [ ! -e $INPUTFILE ]; then
  echo $USAGE;
  exit 1
fi

if [ -z $IPADDR ]; then
  echo $USAGE;
  exit 1
fi

# empty dirs if they exist
rm -rf keys/
rm -rf sections/

# create dirs
mkdir keys
mkdir sections

# quit this if error
(while true; do 
    # generate ssl key
    key=$(openssl rand 16)
    # generate initialization key
    init=$(openssl rand -hex 16)
    # create files
    newUid=$(uuidgen)
    keyfileName="keys/${newUid}.key"
    echo -e "\nregenerating keys"
    echo -n $key > "$keyfileName"
    echo -ne "${IPADDR}/${keyfileName}\n${keyfileName}\n${init}" > enc.keyinfo.temp
    mv enc.keyinfo.temp enc.keyinfo
    sleep 10
done) &

bg_pid=$!
sleep 1

ffmpeg -i $INPUTFILE \
  -filter_complex \
  "[0:v]split=3[v1][v2][v3]; \
  [v1]copy[v1out]; [v2]scale=w=1280:h=720[v2out]; [v3]scale=w=640:h=360[v3out]" \
  -map "[v1out]" -c:v:0 libx264 -x264-params "nal-hrd=cbr:force-cfr=1" -b:v:0 5M -maxrate:v:0 5M -minrate:v:0 5M -bufsize:v:0 10M -preset slow -g 48 -sc_threshold 0 -keyint_min 48 \
  -map "[v2out]" -c:v:1 libx264 -x264-params "nal-hrd=cbr:force-cfr=1" -b:v:1 3M -maxrate:v:1 3M -minrate:v:1 3M -bufsize:v:1 3M -preset slow -g 48 -sc_threshold 0 -keyint_min 48 \
  -map "[v3out]" -c:v:2 libx264 -x264-params "nal-hrd=cbr:force-cfr=1" -b:v:2 1M -maxrate:v:2 1M -minrate:v:2 1M -bufsize:v:2 1M -preset slow -g 48 -sc_threshold 0 -keyint_min 48 \
  -map a:0 -c:a:0 aac -b:a:0 96k -ac 2 \
  -map a:0 -c:a:1 aac -b:a:1 96k -ac 2 \
  -map a:0 -c:a:2 aac -b:a:2 48k -ac 2 \
  -hls_key_info_file enc.keyinfo \
  -hls_time 10 \
  -hls_playlist_type vod \
  -hls_flags independent_segments+periodic_rekey \
  -hls_segment_type mpegts \
  -hls_segment_filename sections/%v_%d.ts \
  -master_pl_name master.m3u8 \
  -var_stream_map "v:0,a:0 v:1,a:1 v:2,a:2" sections/%v.m3u8


python3 scrambler.py -d sections

kill $bg_pid
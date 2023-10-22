#!/usr/bin/env bash
set -euo pipefail


############################################################
# Help                                                     #
############################################################
help()
{
    # Display Help
    echo
    echo "Syntax: $0 [options]"
    echo "options:"
    echo "--src       Source directory"
    echo "--dest      Destiation directory"
    echo
}


############################################################
############################################################
# Main program                                             #
############################################################
############################################################
for i in "$@"; do
    case $i in
        --src=*)
            SRC_DIR="${i#*=}"
            shift
            ;;
        --dest=*)
            DEST_DIR="${i#*=}"
            shift
            ;;
        -*|--*)
            echo "Unknown argument $i" 
            help
            exit 1
            ;;
        *)
            ;;
    esac
done

if [[ -z ${SRC_DIR+x} ]]; then 
    echo "Missing --src option"
    help
    exit 1
fi

if [[ -z ${DEST_DIR+x} ]]; then 
    echo "Missing --desc option"
    help
    exit 1
fi

RESOLUTION=${RESOLUTION:-"640:360"}

for FILE in "${SRC_DIR}"/* ; do
    filename=$(basename $FILE)
    if [[ "$FILE" =~ .*(\.h264)$ ]] ; then
        output=${DEST_DIR}/${filename%.*}.mp4
        set -e
        ffmpeg -loglevel info -n -i "$FILE" -vf "scale=640:360" -crf 30  "${output}"
        if [[ $? -eq 0 ]]; then 
            echo "📹 Convert to MP4 successfully, output: ${output}"
        else
            echo "❌ Could not convert video file ${FILE}; TODO handle the error"
        fi
        set +e
    fi
done

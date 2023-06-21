import av
import sys

def h265ToJpg_demo():
    inputFileName = "D://Testpython//simplest_mediadata_test_sintel.h264"
    container = av.open(inputFileName)
    print("container:", container)
    print("container.streams:", container.streams)
    print("container.format:", container.format)

    for frame in container.decode(video = 0):
        print("process frame: %04d (width: %d, height: %d)" % (frame.index, frame.width, frame.height))
        frame.to_image().save("D://Testpython//output//frame-%04d.jpg" % frame.index)

def main():
    h265ToJpg_demo()

if __name__ == "__main__":
    sys.exit(main())

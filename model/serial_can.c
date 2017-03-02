#include "serial_can.h"
#include <fcntl.h>
#include <termios.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <stdio.h>
#include <errno.h>
#include <sys/ioctl.h>

struct termios oldtio;
int oldio_init = 0;
#define BAUDRATE B115200

int serial_can_open(const char *devname)
{
   int fd;
    struct termios newtio;

    fd = open(devname, O_RDWR | O_NOCTTY ); 
    if (fd <0) {
        fprintf(stderr, "Could not open %s: %s\n", devname, strerror(errno)); 
        return -1; 
    }

    if (oldio_init==0)
    {
        tcgetattr(fd,&oldtio); /* save current port settings */
    }
    oldio_init++;

    bzero(&newtio, sizeof(newtio));
    newtio.c_cflag = CS8 | CLOCAL | CREAD; // hardware flow control
    newtio.c_iflag = IGNPAR;
    newtio.c_oflag = 0;

    /* set input mode (non-canonical, no echo,...) */
    newtio.c_lflag = 0;
    cfsetispeed(&newtio,BAUDRATE);
    cfsetospeed(&newtio,BAUDRATE); 

    newtio.c_cc[VTIME]    = 0;   /* inter-character timer unused */
    newtio.c_cc[VMIN]     = 1;   /* blocking read until 13 chars received */

    tcflush(fd, TCIOFLUSH);
    if (tcsetattr(fd,TCSANOW,&newtio)<0) {
        fprintf(stderr, "Could not set attributes on %s: %s\n", devname, strerror(errno)); 
        close(fd);
        return -1;
    }
    return fd;
}

int serial_can_status(int fd, int *status)
{
    return ioctl(fd, TIOCMGET, &status);
}

int serial_can_send(int fd, const unsigned char data[13])
{
    return write(fd,data,13)==13;
}

int serial_can_recv(int fd, unsigned char data[13])
{
    return read(fd,data,13)==13;
}

void serial_can_close(int fd)
{
    close(fd);
    if (--oldio_init<=0) 
        tcsetattr(fd,TCSANOW,&oldtio);
}

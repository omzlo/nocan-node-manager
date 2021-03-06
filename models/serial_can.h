#ifndef _SERIAL_CAN_H_
#define _SERIAL_CAN_H_

int serial_can_open(const char *devname);

int serial_can_send(int fd, const unsigned char *data);

int serial_can_recv(int fd, char unsigned *data);

void serial_can_close(int fd);

int serial_can_status(int fd, int *status);

#endif

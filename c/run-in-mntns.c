#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#define _GNU_SOURCE
#include <sched.h>
#include <sys/syscall.h>

int main (int argc, char **argv) 
{
    int pid, fd;
    char filename[256];

    if (argc < 3) 
    {
	fprintf (stderr, "Usage: run-in-mntns pid command...\n");
	exit (1);
    }
    pid = atoi(argv[1]);
    snprintf(filename, sizeof(filename), "/proc/%d/ns/mnt", pid);
    fd = open(filename, O_RDONLY);
    if (fd < 0) 
    {
	perror (filename);
	exit (1);
    }
    if (syscall(SYS_setns, fd, CLONE_NEWNS) < 0) 
    {
	perror ("Error setting namespace");
	exit (1);
    }
    close (fd);
    execvp (argv[2], argv + 2);
    perror (argv[2]);
    exit (1);
}

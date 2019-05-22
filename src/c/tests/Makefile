
BUILDDIR=../../../build/tests
CFLAGS = -std=c99 -g -O2 -Wall -Werror

SRCS = $(wildcard *.c)
OBJS = $(patsubst %.c, $(BUILDDIR)/%.o, $(SRCS))
EXE = $(BUILDDIR)/testsuite

all: $(EXE)

$(EXE): $(OBJS)
	$(CC) $(LDFLAGS) -o $@ $(OBJS) $(LIBS)

$(BUILDDIR)/%.o: %.c
	$(CC) $(CFLAGS) -o $@ -c $<

clean:
	rm -f $(OBJS) $(EXE)

relink:
	rm -f $(EXE)
	make all

test: $(EXE)
	- $(EXE)
	rm -rf test_file

retest: relink test
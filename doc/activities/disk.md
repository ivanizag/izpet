# The disk drive

[Back to the activities](README.md)

A floppy disk drive changed what a PET could be used for. A program loaded in
a second or two instead of half a minute, a disk had a directory to see what
was on it, and programs could keep data in files, read and written while they
ran: addresses, accounts, the texts of a word processor. Commodore's drives
connected to the PET by its IEEE-488 port and were small computers of their
own, with their own processor and their own DOS, so the PET only had to send
them commands.

BASIC 4 brought commands for them. This page uses them on a CBM 8032 with an
8050, a drive with two disks of about 500 KB each.

## What you need

Only izpet, started with a disk that does not exist yet, which izpet creates
when the PET writes on it, and a folder of your computer as the second disk:

```bash
mkdir myfolder
go run ./frontend/petebiten -model 8032 -disk work.d80 -disk1 myfolder
```

`-disk` puts a disk image in the drive 0, `-disk1` in the drive 1. The
extension of the file says the drive: `.d80` is an 8050 disk, `.d64` a 4040
one and `.d82` an 8250 one.

The 8032 starts in lowercase, and BASIC takes its commands in lowercase too.

## A new disk

1. **Format the disk with `HEADER "WORK",D0,I01`, and answer `Y`.** A new
   disk had to be formatted before use: `HEADER` gives it a name, `WORK`,
   and an id of two characters, `01`, on the drive 0. BASIC asks first,
   since formatting a disk erases what was on it.

2. **Type `DIRECTORY`.** The directory of the disk, its name and id in reverse
   on top, and the space left in blocks of 254 bytes: 2052 on an empty 8050
   disk.

   ![A new disk](images/disk/header.png)

## Programs

3. **Type a program, save it with `DSAVE "HELLO"`, then `NEW`, `DIRECTORY`,
   `DLOAD "HELLO"` and `RUN`.**

   ```basic
   10 PRINT "HELLO FROM THE DISK"
   DSAVE "HELLO"
   NEW
   DIRECTORY
   DLOAD "HELLO"
   RUN
   ```

   The program is in the directory, a `PRG` file of one block, and `DLOAD`
   brings it back at once.

   ![Saving and loading a program](images/disk/dload.gif)

## A data file

4. **Write a file from a program**, `DOPEN` with `W` to write, `PRINT#` to
   write in it, and `DCLOSE`:

   ```basic
   NEW
   10 DOPEN#1,"NOTES",W
   20 PRINT#1,"BUY MORE DISKS"
   30 DCLOSE#1
   RUN
   ```

5. **Read it back** with `DOPEN` and `INPUT#`:

   ```basic
   NEW
   10 DOPEN#1,"NOTES"
   20 INPUT#1,A$
   30 DCLOSE#1
   40 PRINT A$
   RUN
   ```

   ![A data file read back](images/disk/notes.png)

   `NOTES` is a `SEQ` file, a sequential file, read in the order it was
   written. Mailing lists, accounts and the letters of a word processor were
   kept in files like this one.

## When something goes wrong

6. **Type `DLOAD "NOTHING"`, then `PRINT DS$`.** `DS$` is the status of the
   drive, as its DOS gives it: a number, a message, and the track and sector
   where it happened. 62 is a file not found. After any disk command a
   program could look at `DS` and `DS$` to see whether it worked.

   ![The status of the drive](images/disk/ds.png)

## Keeping the disk tidy

7. **Type `RENAME "HELLO" TO "GREETING"`, `SCRATCH "NOTES"`, answer `Y`, and
   `DIRECTORY`.** `SCRATCH` deletes a file, and says how many it deleted.

   ![A file renamed and one scratched](images/disk/scratch.png)

## A folder of your computer

8. **Load the program and save it in the drive 1, then list it:**

   ```basic
   DLOAD "GREETING"
   DSAVE "GREETING",D1
   DIRECTORY D1
   ```

   The drive 1 is the folder `myfolder`, and the program is in it as
   `greeting.prg`. It is a way to take programs between izpet and your
   computer, or other emulators. Files put in the folder, `.prg` and `.seq`,
   show in the directory.

   ![A folder of the computer as a disk](images/disk/folder.png)

## What next

[The machine language monitor](monitor.md) goes under BASIC, to the
memory and the processor.

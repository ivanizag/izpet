# Writing activities

How the pages in this folder are made, and what makes them good. This is
mainly for the language models that write new activities, and for whoever
reviews their work. Nothing links here on purpose: it is not documentation
for izpet's users. It follows the one of izmac.

## What an activity is

An activity is one thing a PET owner really did, from 1977 to the early
eighties. It is walked through step by step, and the reader sees on the way
what the machine showed back then. Its parts are:

- a page, `doc/activities/<name>.md`;
- an entry on [README.md](README.md), with a title, a picture that links to
  the page, and a paragraph;
- its images, in `doc/activities/images/<name>/`. They are made by running the
  machine, never drawn or edited by hand.

## How to make one

1. **Propose before writing.** List candidate activities with a line each, and
   let the user choose. Write only the ones chosen.
2. **Do the activity on the emulator first**, in the `activities` package,
   and look at every image it makes. Write the text from what the machine
   really does, not from memory of what it did.
3. **Add a function to the generator**: `<name>Screenshots(t *testing.T)` in
   a file of its own, `activities/<name>_test.go`, run from
   `TestActivityScreenshots` in `activities/activities_test.go` with
   `t.Run`. It makes its machine with `testConfig` and `buildTestPET`, and
   drives it with the methods of the PET: `TypeText` for what is typed,
   `PutKey` for a key held, `LoadFile` and `InsertDisk` for the media,
   `RunFrames`, and `waitForText` and `waitForCursor` to wait for the screen.
   Each failure is stopped with `must`.

   Then it takes pictures into an `Album` of the page: `Screenshot`,
   `ScreenshotOf` (which adds a label), `Machine` (the screen over the
   keyboard of the model) or a `Record(...)` recording saved with
   `SaveRecording`.
4. **Generate the images:**

   ```bash
   IZPET_ACTIVITY_SCREENSHOTS=1 go test -count=1 -run 'TestActivityScreenshots/<name>' ./activities
   ```

   Without the variable the test skips itself. It overwrites the images of
   the activities it runs.
5. **Look at every image and every GIF before committing**, every frame of
   the GIFs included. A contact sheet of the frames is the quickest way to
   check. Then delete any image the pages no longer use, and check that every
   image a page names exists.
6. **Commit the generator, the pages and the images together.**

## Software

The ROMs are in izpet. Programs, tapes and disks an activity needs go in the
repository, with where they come from, before the page is written: the
reader has to run what the generator ran. Prefer what can be typed: most PET
software of the time was printed in books and magazines and typed in.

## The page

- **Start with the history.** One or two paragraphs on what this was in its
  day, and why people did it. Then a paragraph on what this page does.
- **"What you need"** says what to download and from where, or that izpet is
  all that is needed.
- **Number the steps once for the whole page.** The numbering continues
  across the `##` sections; it does not restart in each one.
- **Each step starts with the action in bold**, such as "**Start the PET
  2001.**". Then comes what happens, then the picture, then the explanation of
  what the picture shows and why it mattered then.
- **Write for someone who has never seen a PET.** Name a key by the words on
  it, and say how to reach it from a modern keyboard when it is not obvious:
  RUN/STOP is Escape, RVS is F3. The printable characters are typed as they
  are, whatever the layout.
- **Command lines go in `bash` blocks**, followed by a sentence on each
  option that matters.
- **Dates, prices and names are claims too.** Say only what is sure, or leave
  it out.
- **Every claim must be true of what the emulator shows.** Check versions,
  messages, numbers and timings on the machine.
- **Give the real time of anything slow**, such as a tape loading, especially
  when a recording shows it faster.
- **End with "What next"**, linking to the next activity or to the part of
  the README that goes deeper.
- **Start each page with a link back**, `[Back to the activities](README.md)`.

## Screenshots

- **Every picture is made by the generator.** Never use anything local and
  uncommitted.
- **The frame is part of the picture.** `Frame` draws a black border with
  rounded corners, like the glass of the screen, and the screen is shown with
  its pixels in their shape, as the frontend does.
- **Label the frame** with the model, so that each picture says which one it
  is.
- **The picture of a model is the `Machine`**, its screen over its keyboard.
  Use it once per model on a page, and plain screenshots for the rest.
- **Take the picture once the screen is finished**, and with the cursor
  shown: it blinks, `waitForCursor` waits for it.
- **Every run makes the same pictures.** The emulator is deterministic, the
  pictures change only when the emulator or the steps do.
- **Show the result of each important step**, not every step.
- **One picture of one thing.** A GIF that ends on what the next screenshot
  shows makes the screenshot redundant, unless it is used as a thumbnail.

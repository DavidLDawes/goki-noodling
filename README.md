# goki-noodling
Playing with 3D rendering in Golang via the gi library
## History
### JavaScript demo
I wrote a Javascript procedural galaxy generator a decade ago to learn some JS & how to build procedurally driven apps. I also played with some scale modifications - as the scale expands the minor details get dropped, allowing for a very large range of scale.

I learned how to generate the galaxy & tweak star populations to reflect the local Milky Way per Google lookups & wikipedia (ought to add links). I built the (procedural) data out using simple direct sequence random number generators, running the murtmur3 hash on the sector's X, Y & Z to get the seed. I built in a top down, brightest/rarest to dimmest/most common rendering algorithm, so at higher scales you can drop the dimmer stars. 

Using a simple isometric view (looking down from above, so effectively Z is ignored) I built a display with buttons to allow you to pan and scroll. I built a simple 1x1, 2x2, 4x4 doubling zoom model that allows you to get wider and allowed 1/2 & /4 of a sector too. It always uses an (integer power of 2) number of squares along X and Y axes. 
### Go port
Recently I decided to do some Golang UI programming so I can get better at both Golang and UI programming. I built a [Traveler starship designer](https://github.com/DavidLDawes/TravellerBook5SSD) - a simple collection of widgets with text outputs and some logic (calculating tonnage used and remaining, staffing needs, etc.).

Next I wanted to do something more complicated, so I dug up my old JS Galaxy code and ported it to go. Still flat and isometric. Converting data structures and logic was simple. UI coding was where most effort went, the UI APIs are different. Once that was working with an isometric view, I decided I wanted to see if there were any good 3D graphics libraries available for Golang that handle perspective and viewport rendering for 3D sc scenes. I found Gi (link) which has a nice object oriented structured 3D rendering approach and hooks into GPU rendering engines. Just what I was looking for.
### 3D and GI
Ported the code over to th4e gi3d library and got it working, it now dfoes 3D modeling. So far I just build out a 4 x 2 x 1 set of sectors - wide, not too tall, and not deep at all. I've lost the scaling stuff, I'll need to put a variant back in to handle the smooth scaling vs. detail level of rendering to get larger scales.

### Jump Routes
I always wanted to address my main complaint about Traveler: it's 2D. Given the printed tech of the time, that's what was available. Now we have a much richer set of tools available for free.

I added code to loop through the list of stars and create routes - lines from one star to another - if they were close enough in Traveler terms.  

#### Close Enough
I originally built out 6 parsec jump limit. It turns out that at our current density (.003 stars/cubic light year) there are 3,000 stars per 100 x 100 x 100 light year sector. That ends up with the stars being so dense that everything pretty much connects to everything. Not useful as a Traveler map replacement, it's just a bunch of tangled lines.

To make it more useful I dialed back the density some, to maybe 1/3 of reality, and cut the jump limit back to 5. I set the color and intensity and size (remember they are 3D, so the lines have a cross section size) to be larger & brighter for jump 1, scaling down from there. Jumps above 3 are rendered very light and thin - you can see them but you have to look close. The result looks fairly useful as a 3D Traveler map generator, although I continue to tweak it some.

### Jump Networks
The on-line Traveler map has some jump routes on it representing the main traffic lines (and the X boat network). The trade routs largely follow paths that keep jumps to 1 or 2 parsecs. Adding jumps for 1, 2, 3 parsec etc. apart stars as we render the stars allows us to see the routes so I've added that. For the density of starts near us there should be 3,000 stars per 100 x 100 x 100 light year cube (1 million cubic light years). That yields many stars in the 1 to 2 parsec range and rats nests of connections. I've tweaked the density to 1/3 of our local area (~1000 stars/light years cubed) and draw out the J1 through J3 network, except even that can be a bit rats nest-ish, so I limit each star to adding 2 new routes (well the code may be changing rapidly, check it as this is likely out of date). This yields a reasonably pleasant 3D network of random stars.

Added funcs to trace out the network of stars connected to a given star. Process all stars being rendered and highlight the largest connected network.

Add Traveler style world generation per system. While processing stars also check the highest population and tech levels.
## Up Next
Adding an additional panel with a selected star readout (planet details, system details, routes) and buttons to get to other stars connecting to it. First get the text updating dynamically, then get network (lines) and star and routes updating in response to the UI.

# NO RULES
## Interactive Game Script - Episodes 1-4 (Shared Route)
### NoRules Script Format v1

---

@episode 1 "Bad Idea"

@bg malias_bedroom_morning fade

@show malia neutral_phone at center

NARRATOR: There are two types of people in this world.

NARRATOR: People who learn from their mistakes.

NARRATOR: And me.

@sfx phone_buzz

YOU: Easton.

YOU: Of course it's Easton.

@phone show
@text from EASTON: Can we talk? I know you said you needed space but I miss you. I'll be at the park at 8.
@phone hide

@expr malia neutral_stare

YOU: He misses me.

YOU: He missed me three months ago too. Right before he took Jianna to his dad's company gala and I watched the photos on Instagram from my bathroom floor.

YOU: "Can we talk" is the opening line of every conversation that has ever ruined my week.

@expr malia neutral_flat

NARRATOR: Senior year starts today. I was supposed to have a plan.

NARRATOR: The plan was simple: Don't think about Easton King. Don't answer his texts. Don't look at his hands when he talks. Don't remember what it felt like when those hands--

NARRATOR: See, this is the problem.

@expr malia neutral_thinking

NARRATOR: Easton and I ended. Sort of. He couldn't promise me anything because his dad had already promised him to someone else. Jianna Miller. Business merger. Modern-day arranged marriage wrapped in a varsity jacket.

NARRATOR: So I told him I was done waiting.

NARRATOR: That was June.

NARRATOR: It is now September and I have not stopped thinking about him for more than eleven consecutive hours.

NARRATOR: I counted.

@hide malia fade
@bg school_front_morning fade

@show malia neutral_walking at left

NARRATOR: Morhills High. Senior year. Same building. Same hallways. Same boy at the basketball court who pretends he doesn't see me while making sure I see him.

@show josie neutral_excited at right

JOSIE [neutral_excited]: MALIA. First day. Senior year. I need you to tell me I look amazing before I lose my nerve.

MALIA [neutral_smile]: You look amazing.

JOSIE [neutral_relieved]: Okay good because I changed four times and my mom said the first outfit was fine which means it was definitely not fine.

MALIA [neutral_laugh]: Your mom also said my eyebrows were "very thick" at dinner last week.

JOSIE [neutral_laugh]: She meant it as a compliment.

MALIA [neutral_dry]: She did not.

JOSIE [neutral_walking]: So. Any Easton sightings yet?

MALIA [neutral_flat]: No.

JOSIE [neutral_knowing]: You checked.

MALIA [neutral_flat]: I did not check.

JOSIE [neutral_loving]: Babe. You checked the parking lot before you checked your schedule. I saw you.

YOU: I hate having a best friend who actually pays attention.

@hide josie fade
@hide malia fade
@bg school_hallway_morning fade

@show malia neutral_walking at left
@show josie neutral_walking at left

NARRATOR: And then, because the universe has a truly terrible sense of humor--

@show mauricio neutral_flat at right

@expr josie neutral_whisper

JOSIE: Don't look.

MALIA [neutral_flat]: I'm not looking.

JOSIE: You're looking.

MALIA [neutral_annoyed]: I'm looking at the hallway. He happens to be in the hallway. That's geometry, not interest.

@expr mauricio neutral_reading

YOU: Mauricio Reyes.

YOU: Soccer captain. Easton's best friend. My next-door neighbor since I was six. Reads more than anyone I know but would rather die than admit it.

YOU: We don't talk.

YOU: We used to. When we were kids. Then we didn't. Then he started hating me. I never found out why.

YOU: So I hated him back.

YOU: It's been working fine for eight years.

@expr mauricio neutral_glance

YOU: Fine.

YOU: Completely fine.

@hide mauricio fade
@hide josie fade
@hide malia fade

@bg classroom_morning fade

@show mrs_williams neutral_teacher at center
@show malia sitting_neutral at left
@show mauricio sitting_reading at right

MRS_WILLIAMS: Welcome to AP English, seniors. This semester you'll be doing a partner literature project. I'll assign partners Friday.

MRS_WILLIAMS: Oh, grow up. You'll survive.

YOU: Please not him. Anyone but him.

@expr mauricio sitting_eyebrow

YOU: That eyebrow thing he does.

YOU: I don't know what it means. I don't want to know what it means.

YOU: I especially don't want to know why it made my stomach do something inconvenient just now.

@hide mrs_williams fade
@hide malia fade
@hide mauricio fade

@bg cafeteria_noon fade

@show malia sitting_neutral at left
@show josie sitting_neutral at left

@show mark neutral_grin at right

MARK [neutral_grin]: Ladies. Gentlemen. Significant others of all varieties. The king has returned.

JOSIE [neutral_deadpan]: You went to Cabo for two weeks, Mark. Not war.

MARK [neutral_mock_hurt]: Cabo IS war. Do you know what the buffet line looks like at 7 AM?

@move mark to center

MARK [neutral_easy]: Hernandez. You look stressed. First day and you already look like you're planning someone's funeral.

MALIA [neutral_dry]: Yours, if you steal my fries again.

MARK [neutral_stealing_fry]: I would never.

MALIA [neutral_glare]: Mark.

MARK [neutral_innocent]: What? I'm supporting local agriculture.

YOU: Mark Thomas. My best friend since middle school. YouTuber. Professional idiot. Secretly the smartest person in this building but if you told anyone that, he'd deny it so convincingly you'd apologize for the accusation.

YOU: He's also the only person who has never once asked me about Easton.

YOU: I don't know if that's because he doesn't care or because he knows exactly how much I don't want to talk about it.

YOU: With Mark, it could be either.

@show easton neutral_warm at right

YOU: Oh no.

JOSIE: Twelve o'clock. Incoming.

MARK [neutral_quiet]: Want me to make a scene? I can absolutely make a scene.

@choice
  "Let him come." -> EASTON_APPROACH
  "Have Mark make a scene." -> MARK_SCENE
@endchoice

@branch EASTON_APPROACH
  MALIA [neutral_steady]: No. It's fine.

  @move easton to right

  EASTON [neutral_warm]: Hey, Mal.

  MALIA [neutral_flat]: Hey.

  EASTON [neutral_sincere]: You look good. How was your summer?

  MALIA [neutral_controlled]: It was fine.

  YOU: It was not fine. It was three months of pretending my chest didn't hurt every time I drove past the park where we used to sit on the hood of his car and talk about nothing until 2 AM.

  EASTON [neutral_careful]: Did you get my text this morning?

  MALIA [neutral_flat]: I saw it.

  EASTON [neutral_vulnerable]: I meant it. I miss you. I've been thinking--

  MALIA [neutral_cutting]: Easton. Not here.

  @expr easton neutral_soft

  EASTON: Okay. When you're ready.

  @hide easton fade

  YOU: "When you're ready."

  YOU: He always says that. Like it's my timeline holding us back and not his father's merger.

  @gain EASTON_LET_APPROACH
  @affection EASTON +1
@endbranch

@branch MARK_SCENE
  MALIA [neutral_quick]: Yes. Scene. Now.

  MARK [neutral_loud]: JOSIE. JOSIE. IS THAT A SPIDER ON YOUR SHOULDER?

  JOSIE [neutral_screaming]: WHERE? WHERE?!

  MARK [neutral_pointing]: OVER THERE. NO WAIT. OVER THERE.

  @sfx crowd_chaos

  @hide easton fade

  @hide mark fade
  @hide josie fade
  @hide malia fade
  @bg school_hallway_afternoon

  @show malia neutral_exhale at center

  YOU: I'll thank Mark later. With actual fries. An entire plate of them.

  YOU: I'm not ready to see him act like everything can go back to how it was.

  YOU: Because part of me wants it to. And that part is the part I don't trust.

  @gain EASTON_AVOIDED
  @affection MARK +1
@endbranch

@hide malia fade
@hide josie fade
@hide mark fade

@bg gymnasium_afternoon fade

NARRATOR: Afternoon. Basketball tryouts and soccer practice share the gym on the first day because someone in administration hates us personally.

@show malia sitting_watching at left
@show josie sitting_watching at left

JOSIE [neutral_pointing]: Tyler's gotten better over the summer.

MALIA [neutral_flat]: Mm-hm.

JOSIE [neutral_knowing]: You're not looking at Tyler.

MALIA [neutral_defensive]: I'm looking at the general area of athletic activity.

@show elias neutral_stretching at right

YOU: Elias Hall.

YOU: I've known him for two years and I'm not sure I've heard him say more than forty words total.

YOU: But sometimes he looks at people in a way that makes you feel like he just read your entire search history.

@hide elias fade

@show mauricio neutral_dribbling at right

YOU: He wasn't looking at me.

JOSIE [neutral_smirk]: He was looking at you.

MALIA [neutral_fast]: He was looking at the scoreboard.

JOSIE [neutral_dry]: The scoreboard is behind you, Malia.

YOU: Shut up, Josie.

@hide malia fade
@hide josie fade
@hide mauricio fade

@bg malias_house_evening fade

NARRATOR: The thing about having Mauricio Reyes as your next-door neighbor is that your bedroom windows face each other.

NARRATOR: I've had blackout curtains since freshman year.

NARRATOR: He doesn't.

@show malia neutral_looking at center

@sfx arguing_muffled

YOU: It's happening again.

YOU: His parents.

YOU: I've been hearing it through the walls since middle school. His dad. Drunk. His mom. Tired. And Mauricio somewhere in between, being whatever his family needs him to be.

@sfx door_slam

@expr malia neutral_looking_out

@sfx arguing_muffled

YOU: I should close the curtain.

YOU: This isn't my business.

@wait 1.0

YOU: Right.

YOU: None of my business.

@sfx phone_buzz

@phone show
@text from UNKNOWN: nice curtains, Butterfly
@phone hide

YOU: ...

YOU: He hasn't called me that in eight years.

YOU: Butterfly.

YOU: I used to catch butterflies in the yard when we were kids. He'd follow me around and hold the jar. I was six. He was the only person I let help.

YOU: Then one day he stopped helping. And started hating. And I never found out why.

YOU: Until tonight I thought he'd forgotten the word entirely.

@expr malia neutral_stare

NARRATOR: Senior year. Day one. Status: already complicated.

@hide malia fade

@endep

---

@episode 2 "Bad Idea #2"

@bg classroom_morning fade

@timeskip "Three days later"

@show mrs_williams neutral_teacher at center

MRS_WILLIAMS: Partner assignments for the literature project. No switching. No complaining. No bribery -- yes, I'm looking at you, Thomas.

MARK: That was one time and it was a muffin, not a bribe.

MRS_WILLIAMS: Thomas and Uhrin. Garcia and Macari. King and Davis.

MRS_WILLIAMS: Hernandez and...

YOU: Please not him. Please not him.

MRS_WILLIAMS: Reyes.

YOU: Of course.

@show mauricio sitting_reading at right

MALIA: Kill me.

JOSIE: You'll be fine. Just don't stare at his arms when he rolls up his sleeves.

MALIA: I don't stare at his--

JOSIE: You did it yesterday during the assembly. I timed you. Eleven seconds.

MRS_WILLIAMS: You'll choose a novel together, analyze it from dual perspectives, and present your thesis in December. First meeting with your partner by end of this week.

@sfx bell_ring

@hide mrs_williams fade

@expr mauricio standing_flat

@show malia sitting_surprised at left

NARRATOR: His handwriting. Small, precise, nothing wasted.

NARRATOR: It says: "Jane Eyre. My house. Thursday 4pm."

NARRATOR: No question mark. Not a suggestion. A decision he made for both of us.

YOU: The audacity of this man.

YOU: I could refuse. Find him at lunch, throw the paper back, tell him I pick the book and the location.

YOU: But Jane Eyre is actually a good choice.

YOU: And I kind of want to see what his house looks like inside.

YOU: For academic purposes.

YOU: Strictly academic.

@hide mauricio fade
@hide malia fade

@bg school_hallway_afternoon fade

@show malia neutral_locker at left
@show easton neutral_gentle at right

EASTON [neutral_gentle]: Hey. Do you have a second?

MALIA [neutral_flat]: One. Maybe two. Depends on what you say.

EASTON [neutral_earnest]: I talked to my dad last night. About us. About Jianna.

EASTON [neutral_vulnerable]: I told him I don't want the arrangement. He said... he said he needs time to think about it. But he didn't say no. That's new. He always says no.

YOU: He talked to his dad.

YOU: He actually talked to his dad.

YOU: Easton has never -- in two years -- brought me up to Rafferty King. Not once.

EASTON [neutral_hopeful]: I know I've been--I know I haven't been brave. And I know you have no reason to trust me. But I'm trying, Mal. I'm really trying.

@choice
  "That means something to me." -> EASTON_ACKNOWLEDGE
  "Trying isn't the same as doing." -> EASTON_DISTANCE
@endchoice

@branch EASTON_ACKNOWLEDGE
  MALIA [neutral_quiet]: It does mean something.

  @expr easton neutral_relieved

  MALIA [neutral_careful]: But meaning something and being enough are two different things, Easton.

  EASTON [neutral_steady]: I know. I'm not asking you to decide anything. I'm just asking you to know that I'm trying.

  @wait 0.5

  EASTON [neutral_quiet]: You know what my dad said when I told him? He said "You'll outgrow it." Like you were a phase. Like what I feel is something I just haven't been mature enough to stop doing yet.

  EASTON [neutral_direct]: I'm not outgrowing you, Malia. That's the one thing I'm sure of.

  YOU: ...

  YOU: He's never talked about his dad like that before. Never quoted him. Never let me see the bruise.

  YOU: That's not "I'm trying." That's "I'm bleeding and I came here anyway."

  YOU: I don't know what to do with a version of Easton who shows me his wounds instead of promising to heal them.

  @gain EASTON_ACKNOWLEDGED
  @affection EASTON +2
@endbranch

@branch EASTON_DISTANCE
  MALIA [neutral_measured]: I'm glad you talked to him. But talking to your dad once doesn't undo--

  MALIA [neutral_controlled]: I hope it works out for you. I mean that.

  @expr easton neutral_flinch

  EASTON [neutral_quiet]: Yeah. Me too.

  @hide easton fade

  YOU: I said "for you" and not "for us."

  YOU: I did that on purpose.

  YOU: I also kind of wish I hadn't.

  @gain EASTON_DISTANCE
@endbranch

@hide malia fade
@hide easton fade

@bg school_parking_lot_afternoon fade

@show malia neutral_walking at left
@show mauricio neutral_motorcycle at right

MAURICIO [neutral_flat]: Get on.

MALIA [neutral_stopping]: Excuse me?

MAURICIO [neutral_flat]: Thursday is two days away. We should pick a starting point for the project. Get on.

MALIA [neutral_hand_on_hip]: I have a bus to catch.

MAURICIO [neutral_flat]: The bus takes forty minutes. This takes eight.

MALIA [neutral_suspicious]: Why do you care about saving me time?

@expr mauricio neutral_looking

MAURICIO: I don't. I care about saving mine.

@choice
  "[Take the helmet.]" -> TAKE_RIDE
  "I'll take the bus." -> TAKE_BUS
@endchoice

@branch TAKE_RIDE
  @expr malia neutral_helmet

  YOU: Okay.

  YOU: His jacket smells like detergent and something else. Something warm. I'm cataloguing this for no reason.

  @sfx motorcycle_start

  NARRATOR: We don't talk during the ride. But at one point he takes a turn too fast and I grab tighter and I feel his stomach tense under my hands.

  NARRATOR: Neither of us mentions it.

  NARRATOR: When we stop at my house, I get off and hand back the helmet. Our fingers overlap for a second.

  MAURICIO: Same time Thursday.

  MALIA: Yeah.

  @hide mauricio fade

  YOU: My hands still feel warm where they were on his jacket.

  YOU: That's just -- body heat. Basic thermodynamics. I learned that in ninth grade. It doesn't mean--

  YOU: It doesn't mean anything.

  @gain MAURICIO_RIDE
  @affection MAURICIO +1
@endbranch

@branch TAKE_BUS
  MALIA [neutral_stubborn]: I'll take the bus. See you Thursday.

  @expr mauricio neutral_almost_smiling

  MAURICIO: Your call, Butterfly.

  @hide mauricio fade

  YOU: There it is again. Butterfly.

  YOU: He said it like it was nothing. Like it's just a word.

  YOU: It's not just a word. It's the word a ten-year-old boy used to call me while I ran around the yard with a mason jar and he--

  YOU: No.

  YOU: I'm not doing this. I'm going to the bus stop. I'm going to put my earbuds in. I'm going to listen to something loud and angry and not think about the way he almost smiled when he said it.

  YOU: I need to stop using that phrase.

  @gain MAURICIO_REFUSED_RIDE
@endbranch

@hide malia fade

@bg malias_living_room_evening fade

@show malia sitting_couch at left
@show vikki neutral_blunt at right

VIKKI [neutral_blunt]: You've been staring at your phone for twenty minutes. Either text him or put it away.

MALIA [neutral_defensive]: I'm not--

VIKKI [neutral_knowing]: Which one? Basketball boy or angry neighbor?

MALIA [neutral_annoyed]: There is no "which one."

VIKKI [neutral_smirk]: Malia. I have eyes. Also walls in this house are thin and you talk in your sleep.

MALIA [neutral_alarmed]: I do NOT--

VIKKI [neutral_casual]: Last Thursday you said "motorcycle" and then "stop it" and then something in Spanish that I'm choosing to believe was a vocabulary exercise.

YOU: I am going to die.

YOU: I am going to die right here on this couch and Vikki is going to put that on my tombstone.

@show samuel neutral_warm at center

SAMUEL: Girls. Dinner. And Vikki, stop tormenting your sister.

VIKKI [neutral_innocent]: I'm bonding.

@sfx phone_buzz

@phone show
@text from MARK: yo. Elias is being weird. he asked about you today. like, specifically about you. Asked what you like to read.
@phone hide

YOU: Elias asked about me?

YOU: Elias doesn't ask about anyone. He observes. He doesn't ask.

YOU: What does that mean?

@phone show
@text from MARK: idk just thought it was random. or maybe not random. with that dude you can never tell. anyway see you tomorrow
@phone hide

NARRATOR: Day three of senior year. I have an ex who's learning to be brave. A nemesis who's calling me Butterfly. A best friend who steals my fries. And the quietest boy in school just asked about my reading habits.

NARRATOR: So much for a simple year.

@hide malia fade
@hide vikki fade
@hide samuel fade

@endep

---

@episode 3 "Bad Idea #3"

@bg mauricios_bedroom_afternoon fade

NARRATOR: Thursday. 4 PM. The first time I've been inside Mauricio Reyes's house since I was ten.

NARRATOR: It looks different from how I remember. Smaller. Older. The wallpaper in the hallway is peeling. But his room is clean. Extremely clean. Books everywhere -- shelves, desk, floor stacks. Jane Eyre, Wuthering Heights, The Catcher in the Rye. A worn copy of The Unhoneymooners with dog-eared pages.

@show malia standing_looking at left
@show mauricio sitting_desk at right

MALIA [neutral_surprised]: You have more books than the school library.

MAURICIO [neutral_flat]: The school library has seventeen copies of Lord of the Flies and none of Toni Morrison. That's not a library. That's a crime scene.

YOU: He just made a literary joke.

YOU: An actually funny literary joke.

YOU: I'm going to pretend I didn't almost smile.

@expr malia sitting_edge

MAURICIO [neutral_opening_book]: Jane Eyre. Dual perspectives. I take Rochester, you take Jane.

MALIA [neutral_competitive]: I want Rochester.

MAURICIO [neutral_flat]: You are Jane.

MALIA [neutral_narrowing]: What is that supposed to mean?

MAURICIO [neutral_calm]: It means you're stubborn, principled, and you'd rather walk into a storm than admit you need someone to hold the umbrella.

@wait 0.5

YOU: ...

YOU: That is both the most annoying and the most accurate thing anyone has ever said about me.

MALIA [neutral_recovering]: Fine. Jane it is. But only because her chapters are better.

MAURICIO [neutral_almost_smile]: They're not.

NARRATOR: We work for two hours. He doesn't talk much. But when he does, every sentence is about the book and somehow also not about the book.

NARRATOR: "Rochester lies because the truth would cost him everything."

NARRATOR: "Jane doesn't leave because she stops loving him. She leaves because she loves herself more."

NARRATOR: I keep catching myself staring at the way he underlines passages. He uses a ruler. Every line is straight.

@sfx stomach_growl

MAURICIO: When did you last eat?

MALIA [neutral_embarrassed]: That's none of your business.

@hide mauricio fade

@wait 0.5

@show mauricio standing_plate at right

MAURICIO: My mom made extra.

MALIA [neutral_surprised]: You're... feeding me?

MAURICIO [neutral_flat]: You can't analyze nineteenth-century feminist literature on an empty stomach. That's not generosity. That's academic standards.

YOU: He brought me food and framed it as an academic necessity.

YOU: I don't know whether to be annoyed or touched.

YOU: I'm both.

@sfx arguing_muffled

@expr malia sitting_frozen

@expr mauricio standing_jaw_tight

@sfx glass_breaking

@expr mauricio standing_controlled

MAURICIO [neutral_low]: Stay here.

@hide mauricio fade

YOU: I know what just happened downstairs. I've been hearing it through the walls for years.

@wait 1.0

@show mauricio standing_blank at right

YOU: I could pretend I didn't notice. Walk out. Say "see you next week." Keep the wall between us.

YOU: Or I could--

@choice
  "[Say something.]" -> SAY_SOMETHING
  "[Leave without saying anything.]" -> LEAVE_SILENT
@endchoice

@branch SAY_SOMETHING
  MALIA [neutral_quiet]: Mauricio.

  MALIA [neutral_quiet]: You don't have to explain anything. But I want you to know--

  @wait 0.5

  MALIA [neutral_honest]: My parents used to fight too. Before my mom left. I used to sing to my little sister to cover the sound.

  MALIA [neutral_quiet]: I'm not saying I understand your situation. I'm saying I had my own version of it.

  @wait 1.0

  @expr mauricio neutral_quiet

  MAURICIO [neutral_very_quiet]: What did you sing?

  MALIA [neutral_small_smile]: Dusk Till Dawn. Zayn. Every single night for a year.

  @wait 0.5

  MAURICIO [neutral_low]: You should go home, Malia.

  YOU: He used my actual name. Not "Hernandez." Not "Butterfly." My name.

  YOU: It sounded different in his voice than it does in anyone else's.

  MALIA: My window's always open. In case you need somewhere that's not here.

  @hide malia fade
  @hide mauricio fade

  @gain MAURICIO_OPENED_DOOR
  @affection MAURICIO +3
@endbranch

@branch LEAVE_SILENT
  MALIA [neutral_normal]: Okay. Same time next week?

  @expr mauricio neutral_nod

  @hide malia fade
  @hide mauricio fade

  YOU: I didn't say anything.

  YOU: Because what would I say? "Are you okay?" He's clearly not okay.

  YOU: "I'm sorry?" He doesn't want my sorry. He doesn't want anything from me except to finish this project and go back to ignoring each other.

  YOU: But.

  YOU: That butterfly on his keychain.

  YOU: Why does he carry a butterfly?

  @gain MAURICIO_WALL_STAYED
@endbranch

@bg malias_kitchen_evening fade

@show malia standing_cooking at left
@show samuel neutral_warm at center

SAMUEL: So. The Reyes boy. You're doing a project together?

MALIA: Yeah. English literature. Jane Eyre.

SAMUEL [neutral_stirring]: Good book. He seems like a serious kid.

MALIA [neutral_too_fast]: He's fine. It's just a project.

SAMUEL [neutral_knowing_smile]: I didn't say it wasn't.

@sfx phone_buzz

@phone show
@group FRIEND_GROUP
@text from MARK: party at Kelly Baker's tomorrow night. who's in
@text from JOSIE: IN
@text from TYLER: party tomorrow night. who's in
@endgroup
@phone hide

@phone show
@text from MARK: hernandez?
@text from MARK: hernandez
@text from MARK: HERNANDEZ
@text from MARK: I will come to your house
@text to MARK: Fine. I'm in.
@phone hide

@phone show
@text from ELIAS: Hey. Mark gave me your number. Hope that's okay. I wanted to ask -- have you read anything by Sylvia Plath? I'm looking for a recommendation for someone.
@phone hide

YOU: Elias Hall is texting me. About Sylvia Plath. On a Friday night.

YOU: "For someone." Who is "someone"?

YOU: Also, since when does Elias text people? He's the most antisocial person in our entire friend group.

@phone show
@text to ELIAS: The Bell Jar. start there. who's it for?
@text from ELIAS: Myself. I just didn't want to say that and sound pretentious.
@phone hide

YOU: He asked for a recommendation but said "for someone" instead of "for me" because he didn't want to seem pretentious.

YOU: That's either very endearing or very calculated.

YOU: I genuinely cannot tell.

@phone show
@text to ELIAS: not pretentious. good taste actually.
@text from ELIAS: Thanks. Goodnight, Malia.
@phone hide

YOU: "Goodnight, Malia."

YOU: Period at the end and everything. Elias Hall is the only person under 25 who uses periods in texts sincerely.

NARRATOR: Three conversations open. Easton's unread text from this morning. Elias's polite sign-off. And the text from Mauricio: "nice curtains, Butterfly."

@hide malia fade
@hide samuel fade

@endep

---

@episode 4 "Bad Idea #4"

@bg party_night fade

@music party_beat

@show malia neutral_entering at left
@show josie neutral_entering at left

NARRATOR: Kelly Baker's back-to-school party. Seniors only. Which means every single person I'm trying to figure out is in one room.

NARRATOR: This is either going to be fun or a disaster.

NARRATOR: Probably both.

@show mark neutral_waving at right

MARK [neutral_loud]: Hernandez! Beer pong. You and me. Versus Tyler and Josie. Right now.

JOSIE [neutral_competitive]: Oh, you're going down, Thomas.

NARRATOR: Beer pong. Mark and I versus Tyler and Josie.

MARK [neutral_between_shots]: We are a force of nature.

MALIA [neutral_focused]: Shut up and throw.

@sfx crowd_cheer

MARK [neutral_victory]: THAT'S WHAT I'M TALKING ABOUT.

@expr malia neutral_laugh

YOU: Was that--

YOU: No. That was nothing. He was looking at the general area of beer pong activity.

YOU: I need to stop reading his eye movements like they're subtitles.

@hide mark fade
@hide josie fade
@hide malia fade

@bg party_hallway_night

@show malia neutral_frozen at left

@show mauricio neutral_angry at right
@show easton neutral_tense at center

MAURICIO [neutral_low]: --don't do this to her again.

EASTON [neutral_tense]: You don't get to tell me--

MAURICIO [neutral_cutting]: I get to tell you exactly that. Because I'm the one who watches her face every time you pick Jianna. And I'm done watching.

@wait 0.5

EASTON [neutral_dangerous]: What does that mean?

MAURICIO [neutral_flat]: It means figure it out, King. Before someone else does.

@hide easton fade
@expr mauricio neutral_passing

@wait 0.5

@hide mauricio fade

YOU: "Before someone else does."

YOU: What did that mean?

YOU: What did THAT mean?

YOU: Was he talking about me? Was he talking about himself? Was he threatening Easton or warning him?

YOU: My heart is doing something that has nothing to do with the music.

@hide malia fade

@bg party_backyard_night fade

@show malia neutral_sitting at left
@show elias sitting_reading at right

MALIA [neutral_surprised]: You're at a party and you're reading.

ELIAS [neutral_calm]: I started The Bell Jar.

MALIA [neutral_sitting]: Already?

ELIAS [neutral_neutral]: You recommended it twelve hours ago. I don't see the point of waiting.

MALIA [neutral_amused]: And? What do you think so far?

ELIAS [neutral_thoughtful]: I think Esther Greenwood is performing normal the way some people perform confidence. She's not sick yet in chapter three but the reader can already tell because she notices the wrong things.

@wait 0.5

YOU: That's... an incredibly precise reading for someone who started this morning.

MALIA [neutral_interested]: What do you mean "notices the wrong things"?

ELIAS [neutral_calm]: People who are fine notice the big picture. People who are not fine notice cracks. She keeps describing surfaces -- windows, tablecloths, the grain of wood. That's a person who is staring at walls because looking at people costs too much.

@wait 1.0

YOU: He just described something I recognize.

YOU: After my mom left, I spent six months looking at cracks in the ceiling. Fourteen cracks. I counted all of them.

YOU: I've never told anyone that.

YOU: And Elias didn't ask me to. He just said something that made me think about it on my own.

MALIA [neutral_quiet]: That's really good, Elias.

ELIAS [neutral_neutral]: It's just reading.

MALIA [neutral_slight_smile]: No it's not.

@expr elias neutral_shift

ELIAS [neutral_quiet]: You do that too, by the way.

MALIA [neutral_confused]: Do what?

ELIAS [neutral_calm]: Notice cracks. You just hide it better than Esther.

@wait 0.5

@sfx door_open

@show easton neutral_looking at right

EASTON: Mal? You out here?

@expr elias sitting_phone

@gain ELIAS_CRACK
@affection ELIAS +2

@hide elias fade
@hide easton fade
@hide malia fade

@bg party_kitchen_night fade

@show malia neutral_getting_water at left
@show easton neutral_hopeful at right

EASTON [neutral_hopeful]: Can we talk for a minute? Somewhere quiet?

YOU: He looks good tonight. He always looks good. That's part of the problem.

YOU: The other part is that when he says "can we talk," I remember every version of us that almost worked.

EASTON [neutral_sincere]: I just want five minutes. That's all I'm asking.

@show mauricio neutral_flat at center

MAURICIO [neutral_flat]: Don't mind me.

YOU: This kitchen is too small for whatever is happening right now.

@expr mauricio neutral_leaving

MAURICIO [neutral_quiet]: Your butterfly's slipping.

@hide mauricio fade

YOU: He noticed my necklace clasp was coming undone.

YOU: In a dark kitchen. While getting ice. While pretending he wasn't looking at me.

YOU: How does he notice things like that?

EASTON [neutral_careful]: You two have been... talking more?

MALIA [neutral_quick]: We have a project together. English class.

EASTON [neutral_nodding]: Right. The project.

YOU: He doesn't look convinced.

YOU: He shouldn't be.

@hide malia fade
@hide easton fade

@bg party_front_yard_night fade

@show malia sitting_car_hood at center

NARRATOR: I'm sitting in the dark and my phone has four unread conversations.

NARRATOR: Four people who said something today that I'm still thinking about.

@phone show
@text from EASTON: I meant what I said. I'm trying. Whatever you need.
@phone hide

@phone show
@text from MAURICIO: nice curtains, Butterfly
@phone hide

@phone show
@text from MARK: MVP MVP MVP best beer pong partner in history. also you left your jacket inside, I grabbed it, it's in my car.
@phone hide

@phone show
@text from ELIAS: Finished chapter 5. You were right. Goodnight, Malia.
@phone hide

YOU: Easton is trying. I believe that now. The question is whether trying is going to be enough this time.

YOU: Mauricio called me Butterfly and noticed my necklace in the dark and I still haven't replied to his text because I don't know what to say to a boy who hated me for eight years and is now doing things like that.

YOU: Mark grabbed my jacket. Mark always grabs my jacket. Mark remembers every small thing I need before I need it, and somehow I've never once thought about what that means.

YOU: Elias said I notice cracks. And he's right. And the fact that he's right without me ever telling him makes me feel something I can't name.

YOU: Four boys. Four completely different versions of what this year could look like.

YOU: One who's learning to fight for me. One who's been fighting something since before I knew him. One who fights for everyone except himself. And one who doesn't fight at all -- he just sees.

@show josie neutral_soft at right

JOSIE [neutral_soft]: Ready to go?

MALIA: Yeah.

JOSIE [neutral_quiet_knowing]: You okay?

MALIA [neutral_honest]: I don't know.

JOSIE [neutral_arm_around]: That's okay. Not knowing is allowed.

@hide josie fade

YOU: I'm going to respond to one of them.

YOU: Just one.

YOU: The one I respond to first is the one I'm actually thinking about.

YOU: Which means I already know more than I'm admitting.

@choice premium "Route Tendency"
  "Reply to Easton." -> ROUTE_EASTON
  "Reply to Mauricio." -> ROUTE_MAURICIO
  "Reply to Mark." -> ROUTE_MARK
  "Reply to Elias." -> ROUTE_ELIAS
@endchoice

@branch ROUTE_EASTON
  @gain ROUTE_LEAN_EASTON
  @affection EASTON +2

  @phone show
  @text to EASTON: I know you're trying. That matters to me. Can we get coffee this week? Just coffee. Nothing heavy.
  @text from EASTON: Yes. Absolutely yes. Tuesday?
  @text to EASTON: Tuesday.
  @phone hide

  YOU: I chose him once before. Maybe I owe it to both of us to see if this time can be different.

  @bg coffee_shop_morning fade

  @timeskip "Tuesday morning, 7:40 AM"

  NARRATOR: I walk into the coffee shop. He's already there.

  NARRATOR: He's been there for thirty minutes. There are two empty cups in front of him. He drank two coffees waiting for me because he was so nervous about being late that he came early and then didn't know what to do with his hands.

  YOU: He came early.

  YOU: He's never early for anything.

  YOU: But he was early for this.

  @show easton standing_surprised at right

  @expr malia neutral_smile

  YOU: Okay.

  YOU: Maybe this time.
@endbranch

@branch ROUTE_MAURICIO
  @gain ROUTE_LEAN_MAURICIO
  @affection MAURICIO +2

  @phone show
  @text draft: thanks for the food today
  @text delete
  @text draft: your mom's cooking is better than mine
  @text delete
  @text draft: why do you carry a butterfly on your keychain
  @text delete
  @text to MAURICIO: Butterfly is better than Hernandez.
  @phone hide

  @wait 0.5

  @phone show
  @text from MAURICIO: noted.
  @phone hide

  YOU: "Noted."

  YOU: One word. No emoji. Period at the end.

  YOU: But he replied in eight seconds. At midnight. He was awake. He was waiting.

  YOU: Or I'm projecting.

  YOU: But eight seconds.

  @bg malias_bedroom_night fade

  @show malia neutral_looking_out at center

  NARRATOR: I put the phone down and look out my window.

  NARRATOR: His curtain is open. For the first time in a week. His light is on. He's sitting on his bed, reading.

  NARRATOR: He doesn't look over.

  NARRATOR: But his curtain is open.

  YOU: That's not nothing.

  YOU: He heard me. "My window is always open." And his answer isn't words. It's a curtain.

  YOU: Mauricio Reyes doesn't speak in sentences. He speaks in gestures. And I'm starting to learn the language.
@endbranch

@branch ROUTE_MARK
  @gain ROUTE_LEAN_MARK
  @affection MARK +2

  @phone show
  @text to MARK: thanks for grabbing my jacket. MVP of friendship too.
  @text from MARK: always. you want it back tonight or can I use it as a pillow? it smells like your shampoo and honestly that's an upgrade from my actual pillow
  @phone hide

  YOU: ...

  YOU: Did Mark Thomas just say my shampoo smells good?

  YOU: That's -- that's a joke. Obviously. Mark says things like that to everyone.

  @phone show
  @text to MARK: keep it. consider it rent for the beer pong partnership.
  @text from MARK: deal. night Hernandez <3
  @phone hide

  YOU: He sends a heart emoji to everyone.

  YOU: He sends a heart emoji to everyone.

  YOU: ...right?

  @bg malias_house_morning fade

  @timeskip "Sunday morning"

  @show malia neutral_surprised at center

  NARRATOR: I open my front door to go for a run.

  NARRATOR: My jacket is folded on the doorstep. On top of it, a packet of my favorite fries from the place that doesn't open until 6 AM.

  NARRATOR: He went and got fries. At six in the morning. On a Sunday. And left them on my porch without knocking.

  YOU: Mark Thomas has done a thousand kind things for me in six years and I've filed every one under "best friend behavior."

  YOU: So why is this one making my chest do something different?

  YOU: Maybe it's the same thing it always was. Maybe I just never let myself name it.
@endbranch

@branch ROUTE_ELIAS
  @gain ROUTE_LEAN_ELIAS
  @affection ELIAS +2

  @phone show
  @text to ELIAS: chapter 5 is where it starts getting real. wait until chapter 11.
  @text from ELIAS: I'll let you know when I get there. Probably tomorrow.
  @text to ELIAS: do you ever sleep?
  @text from ELIAS: Not when I have a good book.
  @phone hide

  @wait 0.5

  @phone show
  @text from ELIAS: That came out more charming than I intended. I'm genuinely just an insomniac.
  @phone hide

  YOU: He corrected himself.

  YOU: He said something that sounded like a line, realized it, and immediately took it back.

  YOU: I've never met anyone who is that aware of how they sound and that uncomfortable with accidentally being smooth.

  YOU: It's weirdly--

  YOU: No. Not going there tonight.

  @phone show
  @text to ELIAS: goodnight, Elias.
  @text from ELIAS: Goodnight, Malia.
  @phone hide

  YOU: Periods. Both of us. I'm going to think about that longer than I should.

  @bg classroom_morning fade

  @timeskip "Monday morning. AP English."

  @show malia sitting_surprised at center

  NARRATOR: I walk in late.

  NARRATOR: On my desk there is a book. The Bell Jar. A used copy. Certain passages have been underlined in pencil -- light, careful lines.

  NARRATOR: I open to the first underlined passage. It's about Esther noticing the way light falls through a window when everyone else in the room is talking.

  NARRATOR: In the margin, in tiny handwriting: "This one reminded me of you."

  YOU: ...

  YOU: Elias Hall left a book on my desk. With annotations. One of them says "this one reminded me of you" next to a passage about a girl who sees what no one else sees.

  YOU: He is reading me like I am a book. And the terrifying part is I don't think he's wrong about a single underline.
@endbranch

@hide malia fade

@bg black slow

NARRATOR: Senior year. Week one. Four boys. Four open doors.

NARRATOR: I know I can't keep all of them open.

NARRATOR: But tonight, I picked one to lean into.

NARRATOR: And that terrifies me more than any of the bad ideas I've already made.

NARRATOR: -- Your path is taking shape. --

@endep

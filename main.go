package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/tiktoken-go/tokenizer"
	"github.com/urfave/cli/v2"
)

var (
	client         *openai.Client
	OPENAI_API_KEY string
	tictoken       tokenizer.Codec
	modelName      = "gpt-3.5-turbo"
)

const (
	// book
	MetamorphosisURL = "https://www.gutenberg.org/cache/epub/64317/pg64317.txt"
)

func init() {
	runMode := os.Getenv("RUN_MODE")
	if runMode != "dev" {
		log.SetOutput(io.Discard)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	OPENAI_API_KEY = os.Getenv("OPENAI_API_KEY")
	client = openai.NewClient(OPENAI_API_KEY)

	var err error
	tictoken, err = tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		log.Fatal(err)
	}
}

func getTextFromGutenberg(url string) (string, int) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatal(errors.New("failed to fetch book text"))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	bookCompleteText := string(bodyBytes)
	bookCompleteText = strings.Replace(bookCompleteText, "\r", "", -1)

	// Split the text to remove Project Gutenberg's header and footer
	re := regexp.MustCompile(`\*\*\* .+ \*\*\*`)
	split := re.Split(bookCompleteText, -1)

	log.Println("Divided into parts of length:")
	for _, s := range split {
		fmt.Println(len(s))
	}

	// Select the middle of the split, which is the actual book
	if len(split) < 3 {
		log.Fatalln("Expected at least 3 parts after splitting")
	}
	book := split[1]

	// In Go, we can count the number of characters with len(book)
	fmt.Printf("Text contains %d characters\n", len(book))
	_, tokens, err := tictoken.Encode(book)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Text contains %d tokens\n", len(tokens))
	return book, len(tokens)
}

const (
	targetSummarySize = 1000
	modelContextSize  = 4097
	delimiter         = "."
)

func Run(*cli.Context) error {

	brif := NewBrif()
	_ = brif

	log.Println(brif.Config.Version)
	book, numTokens := getTextFromGutenberg(MetamorphosisURL)
	_ = book
	costPerToken := 0.002 / 1000
	price := float64(numTokens) * costPerToken
	fmt.Printf("As of Q1 2023, the approximate price of this summary will somewhere be on the order of: %2f", price)

	summary := summarize(
		book,
		targetSummarySize,
		countSummaryInputSize(targetSummarySize),
		delimiter,
		modelName,
	)

	summary = strings.Replace(summary, "[[[", "", -1)
	summary = strings.Replace(summary, "]]]", "", -1)
	fmt.Println("final summary:\n", summary)

	return nil
}

func countToken(text string) int {
	_, tokens, err := tictoken.Encode(text)
	if err != nil {
		log.Fatal(err)
	}
	return len(tokens)
}
func summarize(text string, targetSummarySize, summaryInputSize int, delimiter, modelName string) string {

	// no need to summarize
	if countToken(text) < targetSummarySize {
		return text
	}

	// do summarize
	if countToken(text) <= summaryInputSize {
		resp, err := gptSummarize(text, targetSummarySize)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("usage: %v", resp.Usage)
		content := resp.Choices[0].Message.Content
		return content

	}

	// too large tokens
	splitInputs := splitTextIntoSections(text, summaryInputSize, delimiter, modelName)
	var summaries []string
	for _, input := range splitInputs {
		summary := summarize(input, targetSummarySize, summaryInputSize, delimiter, modelName)
		summaries = append(summaries, summary)
	}

	newText := strings.Join(summaries, "\n\n")
	return summarize(newText, targetSummarySize, summaryInputSize, delimiter, modelName)

}

func splitTextIntoSections(text string, summaryInputSize int, delimiter, modelName string) []string {
	var (
		section  string
		sections []string
	)

	for len(text) > 0 {
		section, text = takeTokens(text, summaryInputSize, delimiter, modelName)
		sections = append(sections, section)

	}
	return sections
}

func takeTokens(text string, summaryInputSize int, delimiter, modelName string) (string, string) {
	currentTokenCount := countTokensFromMessage(promptMessage("", targetSummarySize), modelName)
	sections := strings.Split(text, delimiter)
	nonEmptySections := make([]string, 0)
	for _, section := range sections {
		if strings.TrimSpace(section) != "" {
			nonEmptySections = append(nonEmptySections, section)
		}
	}

	for i, section := range nonEmptySections {
		if currentTokenCount+countToken(section) >= summaryInputSize {
			// Entering this condition means the incoming section brings us past maxTokenQuantity.

			if i == 0 {
				// If i == 0, then we're in the special case where there exists no divisionPoint-separated
				// section of token length less than summaryInputSize.

				// Thus, we return the first summaryInputSize tokens as a chunk, even if it ends on an
				// awkward split.
				ids, _, err := tictoken.Encode(text)
				if err != nil {
					log.Fatal(err)
				}
				ids = ids[:summaryInputSize]

				maxTokenChunk, err := tictoken.Decode(ids)
				if err != nil {
					log.Fatal(err)
				}
				remainder := text[len(maxTokenChunk):]
				return maxTokenChunk, remainder
			}
			if i == 1 {
				emit := nonEmptySections[0] + delimiter
				remainder := strings.Join(nonEmptySections[1:], delimiter) // TODO exception case
				return emit, remainder
			}
			// Otherwise, return the accumulated text as a chunk.
			emit := strings.Join(nonEmptySections[:i-1], delimiter)
			remainder := strings.Join(nonEmptySections[i-1:], delimiter)
			return emit, remainder

		}
		currentTokenCount += countToken(section + delimiter)
	}
	// means text is not exceeded token limits
	return text, ""

}

func countTokensFromMessage(messages []openai.ChatCompletionMessage, modelName string) int {
	convertToJSONString := func(messages []openai.ChatCompletionMessage) (string, error) {
		jsonData, err := json.Marshal(messages)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	}
	jsonText, err := convertToJSONString(messages)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("TODO: should generate tictoken for each model. currently %s", modelName)

	// count tokens
	_, tokens, _ := tictoken.Encode(jsonText)
	return len(tokens)
}

func countSummaryInputSize(targetSummarySize int) int {

	// Compute the number of tokens that should be used for the context window, the target summary, and the base prompt.
	basePromptSize := countTokensFromMessage(promptMessage("", targetSummarySize), modelName)
	summaryInputSize := modelContextSize - (basePromptSize + targetSummarySize) // can request this size of text to summarize
	return summaryInputSize
}

// https://cli.urfave.org/v2/getting-started/
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	app := &cli.App{
		Name:   "brif",
		Usage:  "summarize novel ",
		Action: Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

package BackTest

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"log"
)

func plotGraph(amount []float64, trade []int) {
	// Create a new plot, err is returned if there was an error
	p := plot.New()
	p.Title.Text = "My Scatter Plot" // Set the title.
	p.X.Label.Text = "Trades"        // Label for X axis.
	p.Y.Label.Text = "Amount"        // Label for Y axis.

	// Create some random points to plot.

	pts := make(plotter.XYs, len(amount))
	for i := 0; i < len(amount); i++ {
		pts[i].Y, pts[i].X = amount[i], float64(trade[i])
	}

	// Make a scatter plotter and add it to the plot.
	s, err := plotter.NewLine(pts)
	if err != nil {
		log.Fatalf("could not create scatter: %v", err)
	}
	p.Add(s)

	// Save the plot to a PNG file.
	if err := p.Save(4*vg.Inch, 4*vg.Inch, "scatter.png"); err != nil {
		log.Fatalf("could not save plot: %v", err)
	}
}

register(alias="front-loaded", evaluator=DecayProfile(lambda x: 1.0 - math.pow(x, 0.1), description="lambda x: 1.0 - math.pow(x, 0.1)"))
register(alias="linear", evaluator=DecayProfile(lambda x: 1.0 - x, description="lambda x: 1.0 - x"))
register(alias="back-loaded", evaluator=DecayProfile(lambda x: 1.0 - math.pow(x, 4.0), description="lambda x: 1.0 - math.pow(x, 4.0)"))
